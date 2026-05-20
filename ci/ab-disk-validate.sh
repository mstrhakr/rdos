#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

OUTPUT_AB="uftc-ab.img"
MIN_SIZE_BYTES=$((28 * 1024 * 1024 * 1024))

usage() {
  cat <<'EOF'
Usage: ./ci/ab-disk-validate.sh [options]

Validates the assembled UFTC A/B + recovery raw disk image.

Options:
  --output PATH          Output raw disk path (default: uftc-ab.img)
  --min-size-bytes N     Minimum logical file size gate (default: 28 GiB)
  -h, --help             Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output)
      [[ $# -lt 2 ]] && { echo "Missing value for --output" >&2; exit 1; }
      OUTPUT_AB="$2"
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

require_cmd sgdisk

if [[ ! -f "$OUTPUT_AB" ]]; then
  echo "Validation failed: A/B disk not found at $OUTPUT_AB" >&2
  exit 1
fi

disk_size="$(stat -c%s "$OUTPUT_AB")"
if (( disk_size < MIN_SIZE_BYTES )); then
  echo "Validation failed: A/B disk logical size $disk_size bytes is below minimum $MIN_SIZE_BYTES bytes" >&2
  exit 1
fi
log "A/B disk size gate passed: $disk_size bytes"

partition_table="$(sgdisk -p "$OUTPUT_AB")"
if [[ "$partition_table" != *"Found valid GPT"* ]]; then
  echo "Validation failed: A/B disk does not contain a valid GPT" >&2
  exit 1
fi

for entry in \
  "1:BIOSBOOT" \
  "2:ESP" \
  "3:ROOT_A" \
  "4:ROOT_B" \
  "5:RECOVERY"; do
  part_number="${entry%%:*}"
  part_name="${entry#*:}"
  part_info="$(sgdisk -i "$part_number" "$OUTPUT_AB")"
  if [[ "$part_info" != *"Partition name: '$part_name'"* ]]; then
    echo "Validation failed: partition $part_number is not labeled $part_name" >&2
    exit 1
  fi
done

log "A/B partition layout check passed: BIOSBOOT, ESP, ROOT_A, ROOT_B, RECOVERY"
log "A/B disk validation complete: $OUTPUT_AB"