#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

INPUT_VHD="RDOS.vhd"
INPUT_DISK=""
INPUT_DISK_ZST=""
OUTPUT_ISO="rdos-installer.iso"
BUILD_MODE="build"
NO_CACHE=0
NO_STAGING=0
STAGING_DIR=""
MIN_SIZE_BYTES=$((700 * 1024 * 1024))
PAYLOAD_LAYOUT="auto"

usage() {
  cat <<'EOF'
Usage: ./ci/iso-build-validate.sh [options]

Builds (or reuses) a RDOS installer ISO through build-installer-iso.sh,
then validates ISO shape, boot assets, and payload integrity.

Options:
  --input-vhd PATH        Input VHD path used by ISO build (default: RDOS.vhd)
  --input-disk PATH       Input raw A/B disk path used by ISO build
  --input-disk-zst PATH   Input compressed A/B payload path used by ISO build
  --output-iso PATH       Output ISO path (default: rdos-installer.iso)
  --mode MODE             build (default) or validate-only
  --payload-layout MODE   Expected payload layout: auto, single, or ab
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
    --input-disk)
      [[ $# -lt 2 ]] && { echo "Missing value for --input-disk" >&2; exit 1; }
      INPUT_DISK="$2"
      shift 2
      ;;
    --input-disk-zst)
      [[ $# -lt 2 ]] && { echo "Missing value for --input-disk-zst" >&2; exit 1; }
      INPUT_DISK_ZST="$2"
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
    --payload-layout)
      [[ $# -lt 2 ]] && { echo "Missing value for --payload-layout" >&2; exit 1; }
      PAYLOAD_LAYOUT="$2"
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

if [[ "$PAYLOAD_LAYOUT" != "auto" && "$PAYLOAD_LAYOUT" != "single" && "$PAYLOAD_LAYOUT" != "ab" ]]; then
  echo "Invalid --payload-layout: $PAYLOAD_LAYOUT (expected auto, single, or ab)" >&2
  exit 1
fi

if ! [[ "$MIN_SIZE_BYTES" =~ ^[0-9]+$ ]]; then
  echo "Invalid --min-size-bytes: $MIN_SIZE_BYTES" >&2
  exit 1
fi

log() {
  printf '[%s] %s\n' "$(date +'%H:%M:%S')" "$1"
}

run_with_heartbeat() {
  local label="$1"
  shift

  "$@" &
  local cmd_pid=$!
  local started_at
  started_at="$(date +%s)"

  while kill -0 "$cmd_pid" 2>/dev/null; do
    sleep 15
    if kill -0 "$cmd_pid" 2>/dev/null; then
      local now elapsed
      now="$(date +%s)"
      elapsed=$((now - started_at))
      log "$label still running (${elapsed}s elapsed)"
    fi
  done

  wait "$cmd_pid"
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

effective_payload_layout="$PAYLOAD_LAYOUT"
if [[ "$effective_payload_layout" == "auto" ]]; then
  if [[ -n "$INPUT_DISK" || -n "$INPUT_DISK_ZST" ]]; then
    effective_payload_layout="ab"
  elif [[ "$BUILD_MODE" == "build" ]]; then
    effective_payload_layout="single"
  else
    effective_payload_layout="ab"
  fi
fi

if [[ "$BUILD_MODE" == "build" ]]; then
  build_args=(--skip-build --output-iso "$OUTPUT_ISO")
  if [[ -n "$INPUT_DISK_ZST" ]]; then
    if [[ ! -f "$INPUT_DISK_ZST" ]]; then
      echo "Input compressed disk not found: $INPUT_DISK_ZST" >&2
      exit 1
    fi
    build_args+=(--input-disk-zst "$INPUT_DISK_ZST")
  elif [[ -n "$INPUT_DISK" ]]; then
    if [[ ! -f "$INPUT_DISK" ]]; then
      echo "Input disk not found: $INPUT_DISK" >&2
      exit 1
    fi
    build_args+=(--input-disk "$INPUT_DISK")
  else
    if [[ ! -f "$INPUT_VHD" ]]; then
      echo "Input VHD not found: $INPUT_VHD" >&2
      echo "Build mode expects a prebuilt VHD or A/B disk image." >&2
      exit 1
    fi
    build_args+=(--input-vhd "$INPUT_VHD")
  fi

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

if ! xorriso -indev "$OUTPUT_ISO" -find /RDOS/RDOS.img.zst -exec report_lba >/dev/null 2>&1; then
  echo "Validation failed: payload /RDOS/RDOS.img.zst not found in ISO" >&2
  exit 1
fi
log "Payload path check passed: /RDOS/RDOS.img.zst"

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

payload_file="$tmpdir/RDOS.img.zst"
log "Extracting compressed payload from ISO"
run_with_heartbeat "Payload extract" xorriso -osirrox on -indev "$OUTPUT_ISO" -extract /RDOS/RDOS.img.zst "$payload_file" >/dev/null

log "Verifying compressed payload integrity"
run_with_heartbeat "zstd integrity test" zstd -t "$payload_file" >/dev/null
log "Compressed payload integrity check passed (zstd -t)"

grub_cfg="$tmpdir/grub.cfg"
isolinux_cfg="$tmpdir/isolinux.cfg"
xorriso -osirrox on -indev "$OUTPUT_ISO" -extract /boot/grub/grub.cfg "$grub_cfg" >/dev/null
xorriso -osirrox on -indev "$OUTPUT_ISO" -extract /syslinux/isolinux.cfg "$isolinux_cfg" >/dev/null

if ! grep -Fq 'RDOS guided installer' "$grub_cfg" || ! grep -Fq 'RDOS automatic install' "$grub_cfg"; then
  echo "Validation failed: GRUB menu does not expose both guided and automatic installer entries" >&2
  exit 1
fi

if ! grep -Fq 'label RDOS_guided' "$isolinux_cfg" || ! grep -Fq 'label RDOS_auto' "$isolinux_cfg"; then
  echo "Validation failed: BIOS menu does not expose both guided and automatic installer entries" >&2
  exit 1
fi

log "Installer menu entry check passed for BIOS and UEFI boot paths"

if [[ "$effective_payload_layout" == "ab" ]]; then
  log "Skipping A/B payload layout check (decompression validation temporarily disabled)"
else
  log "Skipping A/B payload layout check for payload-layout=$effective_payload_layout"
fi

log "ISO validation complete: $OUTPUT_ISO"
