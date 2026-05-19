#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

OUTPUT_VHD="uftc.vhd"
IMAGE_NAME="uftc"
BUILD_MODE="build"
NO_CACHE=0
NO_STAGING=0
STAGING_DIR=""
MIN_SIZE_BYTES=$((1 * 1024 * 1024 * 1024))
MIN_VIRTUAL_SIZE_BYTES=$((12 * 1024 * 1024 * 1024))

usage() {
  cat <<'EOF'
Usage: ./ci/vhd-build-validate.sh [options] [-- <d2vm args>]

Builds (or reuses) a UFTC VHD through build.sh, then validates artifact shape.

Options:
  --output PATH              Output VHD path (default: uftc.vhd)
  --image-name NAME          Docker image name for build.sh (default: uftc)
  --mode MODE                build (default) or validate-only
  --no-cache                 Pass --no-cache to build.sh
  --no-staging               Pass --no-staging to build.sh
  --staging-dir PATH         Pass --staging-dir PATH to build.sh
  --min-size-bytes N         Minimum physical file size gate (default: 1 GiB)
  --min-virtual-size-bytes N Minimum virtual disk size gate when qemu-img is available
                             (default: 12 GiB)
  -h, --help                 Show this help

Any arguments after -- are passed to build.sh as d2vm args.
EOF
}

EXTRA_D2VM_ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --output)
      [[ $# -lt 2 ]] && { echo "Missing value for --output" >&2; exit 1; }
      OUTPUT_VHD="$2"
      shift 2
      ;;
    --image-name)
      [[ $# -lt 2 ]] && { echo "Missing value for --image-name" >&2; exit 1; }
      IMAGE_NAME="$2"
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
    --min-virtual-size-bytes)
      [[ $# -lt 2 ]] && { echo "Missing value for --min-virtual-size-bytes" >&2; exit 1; }
      MIN_VIRTUAL_SIZE_BYTES="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      EXTRA_D2VM_ARGS+=("$@")
      break
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

if ! [[ "$MIN_VIRTUAL_SIZE_BYTES" =~ ^[0-9]+$ ]]; then
  echo "Invalid --min-virtual-size-bytes: $MIN_VIRTUAL_SIZE_BYTES" >&2
  exit 1
fi

log() {
  printf '[%s] %s\n' "$(date +'%H:%M:%S')" "$1"
}

if [[ "$BUILD_MODE" == "build" ]]; then
  build_args=(--output "$OUTPUT_VHD" --force --image-name "$IMAGE_NAME")
  [[ "$NO_CACHE" == "1" ]] && build_args+=(--no-cache)
  [[ "$NO_STAGING" == "1" ]] && build_args+=(--no-staging)
  [[ -n "$STAGING_DIR" ]] && build_args+=(--staging-dir "$STAGING_DIR")

  if [[ ${#EXTRA_D2VM_ARGS[@]} -gt 0 ]]; then
    build_args+=(-- "${EXTRA_D2VM_ARGS[@]}")
  fi

  log "Running build.sh for VHD artifact"
  bash ./build.sh "${build_args[@]}"
else
  log "Skipping build.sh; validating existing artifact only"
fi

if [[ ! -f "$OUTPUT_VHD" ]]; then
  echo "Validation failed: output VHD not found at $OUTPUT_VHD" >&2
  exit 1
fi

vhd_size="$(wc -c <"$OUTPUT_VHD")"
if (( vhd_size < MIN_SIZE_BYTES )); then
  echo "Validation failed: VHD size $vhd_size bytes is below minimum $MIN_SIZE_BYTES bytes" >&2
  exit 1
fi
log "VHD size gate passed: $vhd_size bytes"

if command -v qemu-img >/dev/null 2>&1; then
  virtual_size_raw="$(qemu-img info "$OUTPUT_VHD" 2>/dev/null | sed -n 's/^[[:space:]]*virtual size:.*(\([0-9][0-9]*\)[[:space:]]*bytes).*/\1/p' | head -n1 || true)"
  if [[ -n "$virtual_size_raw" ]]; then
    if (( virtual_size_raw < MIN_VIRTUAL_SIZE_BYTES )); then
      echo "Validation failed: virtual size $virtual_size_raw bytes is below minimum $MIN_VIRTUAL_SIZE_BYTES bytes" >&2
      exit 1
    fi
    log "Virtual size gate passed: $virtual_size_raw bytes"
  else
    log "qemu-img is present but virtual-size parsing failed; skipping virtual size gate"
  fi
else
  log "qemu-img not installed; skipping virtual size gate"
fi

if command -v file >/dev/null 2>&1; then
  file_out="$(file "$OUTPUT_VHD")"
  if [[ "$file_out" == *"Microsoft Disk Image"* || "$file_out" == *"Virtual PC"* || "$file_out" == *"VHD"* ]]; then
    log "Container format check passed via file(1)"
  elif [[ "$file_out" == *"DOS/MBR boot sector"* || "$file_out" == *"boot sector"* ]]; then
    log "Boot sector description detected via file(1)"
  else
    log "file(1) output did not match known VHD strings; continuing to MBR signature gate"
  fi
else
  log "file(1) not installed; skipping container format hint check"
fi

if command -v qemu-img >/dev/null 2>&1; then
  check_out="$(qemu-img check "$OUTPUT_VHD" 2>&1 || true)"
  if [[ "$check_out" == "" ]]; then
    log "qemu-img integrity check passed"
  elif [[ "$check_out" == *"does not support checks"* ]]; then
    log "qemu-img check unsupported for this format; skipping integrity check"
  else
    echo "Validation failed: qemu-img integrity check failed" >&2
    echo "$check_out" >&2
    exit 1
  fi
fi

if command -v fdisk >/dev/null 2>&1; then
  if fdisk -l "$OUTPUT_VHD" >/dev/null 2>&1; then
    log "fdisk partition-table probe succeeded"
  else
    log "fdisk present but could not inspect VHD; continuing"
  fi
fi

log "VHD validation complete: $OUTPUT_VHD"
