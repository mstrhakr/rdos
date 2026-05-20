#!/usr/bin/env bash
set -euo pipefail

ACTION="install"
INCLUDE_DOCKER=1
START_DOCKER=0
ENABLE_DOCKER=0

usage() {
  cat <<'EOF'
Usage: ./setup-build-deps.sh [options]

Installs or reports the host-side dependencies needed for the full UFTC build
pipeline, including A/B disk assembly, ISO generation, and CI preflight checks.

Options:
  --print-only       Print the packages that would be installed without changing the system
  --skip-docker      Do not install a Docker package
  --start-docker     Start the Docker service after install when supported
  --enable-docker    Enable the Docker service at boot when supported
  -h, --help         Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --print-only)
      ACTION="print"
      shift
      ;;
    --skip-docker)
      INCLUDE_DOCKER=0
      shift
      ;;
    --start-docker)
      START_DOCKER=1
      shift
      ;;
    --enable-docker)
      ENABLE_DOCKER=1
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

log() {
  printf '[setup-build-deps] %s\n' "$1"
}

detect_pkg_manager() {
  if command -v apt-get >/dev/null 2>&1; then
    echo apt
  elif command -v dnf >/dev/null 2>&1; then
    echo dnf
  else
    echo unsupported
  fi
}

install_with_apt() {
  local packages=(
    bash
    curl
    dosfstools
    e2fsprogs
    file
    gdisk
    grub-common
    grub-efi-amd64-bin
    grub-pc-bin
    parted
    qemu-utils
    rsync
    shellcheck
    util-linux
    xorriso
    zstd
  )

  if [[ "$INCLUDE_DOCKER" == "1" ]]; then
    packages+=(docker.io)
  fi

  if [[ "$ACTION" == "print" ]]; then
    printf 'apt-get install -y'
    printf ' %s' "${packages[@]}"
    printf '\n'
    return 0
  fi

  sudo apt-get update
  sudo apt-get install -y "${packages[@]}"
}

install_with_dnf() {
  local packages=(
    bash
    curl
    dosfstools
    e2fsprogs
    file
    gdisk
    grub2-efi-x64-modules
    grub2-pc-modules
    grub2-tools-extra
    parted
    qemu-img
    rsync
    ShellCheck
    util-linux
    xorriso
    zstd
  )

  if [[ "$ACTION" == "print" ]]; then
    printf 'dnf install -y'
    printf ' %s' "${packages[@]}"
    if [[ "$INCLUDE_DOCKER" == "1" ]]; then
      printf ' %s' moby-engine
    fi
    printf '\n'
    return 0
  fi

  sudo dnf install -y "${packages[@]}"
  if [[ "$INCLUDE_DOCKER" == "1" ]]; then
    sudo dnf install -y moby-engine || sudo dnf install -y docker
  fi
}

maybe_manage_docker_service() {
  if [[ "$INCLUDE_DOCKER" != "1" ]]; then
    return 0
  fi

  if ! command -v systemctl >/dev/null 2>&1; then
    return 0
  fi

  if ! systemctl list-unit-files docker.service >/dev/null 2>&1; then
    return 0
  fi

  if [[ "$ENABLE_DOCKER" == "1" ]]; then
    sudo systemctl enable docker.service
  fi

  if [[ "$START_DOCKER" == "1" ]]; then
    sudo systemctl start docker.service
  fi
}

pkg_manager="$(detect_pkg_manager)"
case "$pkg_manager" in
  apt)
    log "Using apt-get package mapping"
    install_with_apt
    ;;
  dnf)
    log "Using dnf package mapping"
    install_with_dnf
    ;;
  *)
    echo "Unsupported package manager. Supported: apt-get, dnf" >&2
    exit 1
    ;;
esac

maybe_manage_docker_service

if [[ "$ACTION" == "print" ]]; then
  log "Printed package install command only"
else
  log "Build dependency setup complete"
  if [[ "$INCLUDE_DOCKER" == "1" ]]; then
    log "If Docker is newly installed, verify your user/service configuration before running builds"
  fi
fi
