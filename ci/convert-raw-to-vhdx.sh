#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

INPUT_RAW="rdos-prod.raw"
OUTPUT_VHDX="rdos.vhdx"
FORCE_OVERWRITE=0

usage() {
  cat <<'EOF'
Usage: ./ci/convert-raw-to-vhdx.sh [options]

Converts a raw disk artifact to VHDX for Hyper-V development workflows.

Options:
  --input PATH     Input raw disk path (default: rdos-prod.raw)
  --output PATH    Output VHDX path (default: rdos.vhdx)
  -f, --force      Overwrite output when it already exists
  -h, --help       Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input)
      [[ $# -lt 2 ]] && { echo "Missing value for --input" >&2; exit 1; }
      INPUT_RAW="$2"
      shift 2
      ;;
    --output)
      [[ $# -lt 2 ]] && { echo "Missing value for --output" >&2; exit 1; }
      OUTPUT_VHDX="$2"
      shift 2
      ;;
    -f|--force)
      FORCE_OVERWRITE=1
      shift
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

if ! command -v qemu-img >/dev/null 2>&1; then
  echo "Missing required command: qemu-img" >&2
  exit 1
fi

if [[ ! -f "$INPUT_RAW" ]]; then
  echo "Input raw image not found: $INPUT_RAW" >&2
  exit 1
fi

if [[ -f "$OUTPUT_VHDX" ]] && [[ "$FORCE_OVERWRITE" != "1" ]]; then
  echo "Output already exists: $OUTPUT_VHDX" >&2
  echo "Use --force to overwrite, or choose a different --output path." >&2
  exit 1
fi

if [[ -f "$OUTPUT_VHDX" ]]; then
  if ! rm -f "$OUTPUT_VHDX" 2>/dev/null; then
    output_win_path=""
    if command -v wslpath >/dev/null 2>&1; then
      output_win_path="$(wslpath -w "$OUTPUT_VHDX" 2>/dev/null || true)"
    fi

    if [[ -n "$output_win_path" ]] && command -v cmd.exe >/dev/null 2>&1; then
      cmd.exe /C "del /F /Q \"$output_win_path\"" >/dev/null 2>&1 || true
    fi

    if [[ -f "$OUTPUT_VHDX" ]]; then
      echo "Unable to remove existing output: $OUTPUT_VHDX" >&2
      echo "The file is likely locked by Hyper-V or blocked by Windows ACLs." >&2
      echo "Detach the disk from any VM, run fixPerms.cmd, then retry." >&2
      exit 1
    fi
  fi
fi

printf '[%s] Converting %s -> %s\n' "$(date +'%H:%M:%S')" "$INPUT_RAW" "$OUTPUT_VHDX"
qemu-img convert -f raw -O vhdx "$INPUT_RAW" "$OUTPUT_VHDX"
printf '[%s] VHDX ready: %s\n' "$(date +'%H:%M:%S')" "$OUTPUT_VHDX"
