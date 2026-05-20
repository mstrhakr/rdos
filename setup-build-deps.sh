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
  --print-only       Print the packages and Docker install flow without changing the system
  --skip-docker      Do not install Docker
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

docker_repo_info_apt() {
  . /etc/os-release

  # Determine codename safely
  local codename=""
  if [[ -n "${UBUNTU_CODENAME:-}" ]]; then
    codename="$UBUNTU_CODENAME"
  elif [[ -n "${VERSION_CODENAME:-}" ]]; then
    codename="$VERSION_CODENAME"
  fi

  case "$ID" in
    ubuntu)
      if [[ -z "$codename" ]]; then
        echo "ERROR: Could not determine Ubuntu codename (UBUNTU_CODENAME or VERSION_CODENAME not set in /etc/os-release)" >&2
        return 1
      fi
      printf '%s\n' "https://download.docker.com/linux/ubuntu $codename"
      ;;
    debian)
      if [[ -z "$codename" ]]; then
        echo "ERROR: Could not determine Debian codename (VERSION_CODENAME not set in /etc/os-release)" >&2
        return 1
      fi
      printf '%s\n' "https://download.docker.com/linux/debian $codename"
      ;;
    *)
      return 1
      ;;
  esac
}

sanitize_docker_sources_apt() {
  . /etc/os-release

  local expected_path=""
  case "$ID" in
    ubuntu)
      expected_path="linux/ubuntu"
      ;;
    debian)
      expected_path="linux/debian"
      ;;
    *)
      return 0
      ;;
  esac

  local file
  local changed=0

  for file in /etc/apt/sources.list.d/docker.list /etc/apt/sources.list.d/docker.sources; do
    if [[ ! -f "$file" ]]; then
      continue
    fi
    if ! grep -q "download.docker.com" "$file"; then
      continue
    fi
    if ! grep -q "$expected_path" "$file"; then
      log "Removing stale Docker apt source: $file"
      sudo rm -f "$file"
      changed=1
    fi
  done

  if grep -q "download.docker.com" /etc/apt/sources.list 2>/dev/null; then
    if ! grep -q "$expected_path" /etc/apt/sources.list 2>/dev/null; then
      log "Warning: mismatched Docker source found in /etc/apt/sources.list; remove or fix it if apt update fails"
    fi
  fi

  if [[ "$changed" -eq 1 ]]; then
    log "Removed stale Docker source entries for this host distro"
  fi
}

install_docker_official_apt() {
  local repo_info
  local repo_url
  local repo_codename

  repo_info="$(docker_repo_info_apt)" || return 1
  repo_url="${repo_info% *}"
  repo_codename="${repo_info#* }"

  sanitize_docker_sources_apt

  sudo apt-get remove -y docker.io docker-compose docker-compose-v2 docker-doc podman-docker containerd runc >/dev/null 2>&1 || true
  sudo apt-get update
  sudo apt-get install -y ca-certificates curl
  sudo install -m 0755 -d /etc/apt/keyrings
  sudo curl -fsSL "$repo_url/gpg" -o /etc/apt/keyrings/docker.asc
  sudo chmod a+r /etc/apt/keyrings/docker.asc
  sudo tee /etc/apt/sources.list.d/docker.sources >/dev/null <<EOF
Types: deb
URIs: $repo_url
Suites: $repo_codename
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
EOF
  sudo apt-get update
  sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
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

  if [[ "$ACTION" == "print" ]]; then
    printf 'apt-get install -y'
    printf ' %s' "${packages[@]}"
    printf '\n'
    if [[ "$INCLUDE_DOCKER" == "1" ]]; then
      cat <<'DOCKER_INSTALL_EOF'
# Docker official repo install if docker is not already present:
sudo apt-get remove -y docker.io docker-compose docker-compose-v2 docker-doc podman-docker containerd runc || true
sudo apt-get update
sudo apt-get install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
CODENAME=""
if [ -n "${UBUNTU_CODENAME:-}" ]; then CODENAME="$UBUNTU_CODENAME"; elif [ -n "${VERSION_CODENAME:-}" ]; then CODENAME="$VERSION_CODENAME"; fi
if [ -z "$CODENAME" ]; then echo "ERROR: Could not determine Ubuntu codename (UBUNTU_CODENAME or VERSION_CODENAME not set in /etc/os-release)" >&2; exit 1; fi
sudo tee /etc/apt/sources.list.d/docker.sources >/dev/null <<EOF
Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: $CODENAME
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
EOF
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
DOCKER_INSTALL_EOF
    fi
    return 0
  fi

  sanitize_docker_sources_apt
  sudo apt-get update
  sudo apt-get install -y "${packages[@]}"

  if [[ "$INCLUDE_DOCKER" == "1" ]]; then
    if command -v docker >/dev/null 2>&1; then
      log "Docker already present; skipping Docker repo install"
    else
      log "Installing Docker from the official Docker apt repository"
      if ! install_docker_official_apt; then
        log "Official Docker repo install unsupported on this host; falling back to docker.io"
        sudo apt-get install -y docker.io
      fi
    fi
  fi
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
