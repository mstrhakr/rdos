#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

INPUT_VHD="uftc.vhd"
OUTPUT_ISO="uftc-installer.iso"
BUILD_MODE="build"
NO_CACHE=0
NO_STAGING=0
STAGING_DIR=""
MIN_SIZE_BYTES=$((700 * 1024 * 1024))

usage() {
  cat <<'EOF'
Usage: ./ci/iso-build-validate.sh [options]

Builds (or reuses) a UFTC installer ISO through build-installer-iso.sh,
then validates ISO shape, boot assets, and payload integrity.

Options:
  --input-vhd PATH        Input VHD path used by ISO build (default: uftc.vhd)
  --output-iso PATH       Output ISO path (default: uftc-installer.iso)
  --mode MODE             build (default) or validate-only
  --no-cache              Pass --no-cache to build-installer-iso.sh
  --no-staging            Pass --no-staging to build-installer-iso.sh
  --staging-dir PATH      Pass --staging-dir PATH to build-installer-iso.sh
  --min-size-bytes N      Minimum ISO file size gate (default: 700 MiB)
  -h, --help              Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input-vhd)
      [[ $# -lt 2 ]] && { echo "Missing value for --input-vhd" >&2; exit 1; }
      INPUT_VHD="$2"
      shift 2
      ;;
    --output-iso)
      [[ $# -lt 2 ]] && { echo "Missing value for --output-iso" >&2; exit 1; }
      OUTPUT_ISO="$2"
      shift 2
      ;;
    --mode)
      [[ $# -lt 2 ]] && { echo "Missing value for --mode" >&2; exit 1; }
      BUILD_MODE="$2"
      shift 2
      ;;
    --no-cache)
      NO_CACHE=1
      shift
      ;;
    --no-staging)
      NO_STAGING=1
      shift
      ;;
    --staging-dir)
      [[ $# -lt 2 ]] && { echo "Missing value for --staging-dir" >&2; exit 1; }
      STAGING_DIR="$2"
      shift 2
      ;;
    --min-size-bytes)
      [[ $# -lt 2 ]] && { echo "Missing value for --min-size-bytes" >&2; exit 1; }
      MIN_SIZE_BYTES="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ "$BUILD_MODE" != "build" && "$BUILD_MODE" != "validate-only" ]]; then
  echo "Invalid --mode: $BUILD_MODE (expected build or validate-only)" >&2
  exit 1
fi

if ! [[ "$MIN_SIZE_BYTES" =~ ^[0-9]+$ ]]; then
  echo "Invalid --min-size-bytes: $MIN_SIZE_BYTES" >&2
  exit 1
fi

log() {
  printf '[%s] %s\n' "$(date +'%H:%M:%S')" "$1"
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Missing required command: $cmd" >&2
    exit 1
  fi
}

require_cmd xorriso
require_cmd zstd

if [[ "$BUILD_MODE" == "build" ]]; then
  if [[ ! -f "$INPUT_VHD" ]]; then
    echo "Input VHD not found: $INPUT_VHD" >&2
    echo "Build mode expects a prebuilt VHD. Run ci/vhd-build-validate.sh first." >&2
    exit 1
  fi

  build_args=(--skip-build --input-vhd "$INPUT_VHD" --output-iso "$OUTPUT_ISO")
  [[ "$NO_CACHE" == "1" ]] && build_args+=(--no-cache)
  [[ "$NO_STAGING" == "1" ]] && build_args+=(--no-staging)
  [[ -n "$STAGING_DIR" ]] && build_args+=(--staging-dir "$STAGING_DIR")

  log "Running build-installer-iso.sh"
  bash ./build-installer-iso.sh "${build_args[@]}"
else
  log "Skipping build-installer-iso.sh; validating existing ISO only"
fi

if [[ ! -f "$OUTPUT_ISO" ]]; then
  echo "Validation failed: output ISO not found at $OUTPUT_ISO" >&2
  exit 1
fi

iso_size="$(wc -c <"$OUTPUT_ISO")"
if (( iso_size < MIN_SIZE_BYTES )); then
  echo "Validation failed: ISO size $iso_size bytes is below minimum $MIN_SIZE_BYTES bytes" >&2
  exit 1
fi
log "ISO size gate passed: $iso_size bytes"

if command -v file >/dev/null 2>&1; then
  file_out="$(file "$OUTPUT_ISO")"
  if [[ "$file_out" == *"ISO 9660"* ]]; then
    log "ISO format check passed via file(1)"
  else
    echo "Validation failed: file(1) did not identify an ISO 9660 image" >&2
    echo "$file_out" >&2
    exit 1
  fi
fi

if ! xorriso -indev "$OUTPUT_ISO" -find /uftc/uftc.img.zst -exec report_lba >/dev/null 2>&1; then
  echo "Validation failed: payload /uftc/uftc.img.zst not found in ISO" >&2
  exit 1
fi
log "Payload path check passed: /uftc/uftc.img.zst"

if ! xorriso -indev "$OUTPUT_ISO" -find /syslinux/isolinux.bin -exec report_lba >/dev/null 2>&1; then
  echo "Validation failed: BIOS boot asset /syslinux/isolinux.bin not found" >&2
  exit 1
fi
log "BIOS boot asset check passed"

if xorriso -indev "$OUTPUT_ISO" -find /EFI/boot/bootx64.efi -exec report_lba >/dev/null 2>&1; then
  :
elif xorriso -indev "$OUTPUT_ISO" -find /EFI/BOOT/BOOTX64.EFI -exec report_lba >/dev/null 2>&1; then
  :
elif xorriso -indev "$OUTPUT_ISO" -find /EFI/BOOT/BOOTx64.EFI -exec report_lba >/dev/null 2>&1; then
  :
else
  echo "Validation failed: UEFI boot asset BOOTx64.EFI not found" >&2
  exit 1
fi
log "UEFI boot asset check passed"

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

payload_file="$tmpdir/uftc.img.zst"
xorriso -osirrox on -indev "$OUTPUT_ISO" -extract /uftc/uftc.img.zst "$payload_file" >/dev/null
zstd -t "$payload_file" >/dev/null
log "Compressed payload integrity check passed (zstd -t)"

log "ISO validation complete: $OUTPUT_ISO"
