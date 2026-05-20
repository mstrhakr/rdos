#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SUDOERS_FILE="/etc/sudoers.d/uftc-build-nopasswd"
ACTION="install"
TARGET_USER="${SUDO_USER:-${USER:-}}"

usage() {
  cat <<'EOF'
Usage: ./setup-build-sudoers.sh [options]

Installs or removes a local sudoers rule for UFTC build commands so build scripts
can run without repeatedly prompting for a sudo password.

Options:
  --user NAME     Linux username for sudoers rule (default: current user)
  --remove        Remove the sudoers rule
  -h, --help      Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --user)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      TARGET_USER="$2"
      shift 2
      ;;
    --remove)
      ACTION="remove"
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

if [[ -z "$TARGET_USER" ]]; then
  echo "Could not determine target user. Use --user NAME." >&2
  exit 1
fi

if [[ "$ACTION" == "remove" ]]; then
  sudo rm -f "$SUDOERS_FILE"
  echo "Removed sudoers rule: $SUDOERS_FILE"
  exit 0
fi

DOCKER_BIN="$(command -v docker || true)"
if [[ -z "$DOCKER_BIN" ]]; then
  echo "docker command not found in PATH." >&2
  exit 1
fi

D2VM_PATH="$SCRIPT_DIR/d2vm"
if [[ ! -f "$D2VM_PATH" ]]; then
  echo "Expected d2vm wrapper not found at: $D2VM_PATH" >&2
  exit 1
fi

BUILD_AB_DISK_PATH="$SCRIPT_DIR/build-ab-disk.sh"
if [[ ! -f "$BUILD_AB_DISK_PATH" ]]; then
  echo "Expected A/B assembly script not found at: $BUILD_AB_DISK_PATH" >&2
  exit 1
fi

tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT

cat >"$tmp_file" <<EOF
# Managed by setup-build-sudoers.sh for UFTC local build helpers.
# Remove with: ./setup-build-sudoers.sh --remove
# Only exact pipeline commands are authorized.
$TARGET_USER ALL=(root) NOPASSWD: $DOCKER_BIN build . -t *
$TARGET_USER ALL=(root) NOPASSWD: $DOCKER_BIN build --no-cache . -t *
$TARGET_USER ALL=(root) NOPASSWD: $D2VM_PATH convert * -o * --bootloader grub --boot-size * --size * --network-manager none *
$TARGET_USER ALL=(root) NOPASSWD: $BUILD_AB_DISK_PATH --prod-vhd * --recovery-vhd * --output *
EOF

sudo install -m 0440 "$tmp_file" "$SUDOERS_FILE"
sudo visudo -cf "$SUDOERS_FILE" >/dev/null

echo "Installed sudoers rule: $SUDOERS_FILE"
echo "User: $TARGET_USER"
echo "Allowed commands without password:"
echo "  $DOCKER_BIN build . -t <image>"
echo "  $DOCKER_BIN build --no-cache . -t <image>"
echo "  $D2VM_PATH convert <image> -o <output> --bootloader grub --boot-size <size> --size <size> --network-manager none [extra args]"
echo "  $BUILD_AB_DISK_PATH --prod-vhd <path> --recovery-vhd <path> --output <path>"
