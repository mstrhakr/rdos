#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

MODE="local"
PATH_SCOPE="preflight"
SHELLCHECK_SEVERITY=""

usage() {
  cat <<'EOF'
Usage: ./ci/preflight-validate.sh [--mode local|ci] [--path preflight|vhd|iso|all] [--shellcheck-severity error|warning|info|style]

Preflight validation checks for UFTC:
- Bash syntax checks for key scripts
- Strict mode guards in build entry scripts
- Help-path parsing for build scripts
- CRLF detection on Linux-consumed scripts
- Executable-bit checks for key entrypoints
- Host command presence checks for later build paths

Options:
  --mode MODE    Validation mode: local (default) or ci
  --path SCOPE   Target path: preflight (default), vhd, iso, or all
  --shellcheck-severity LEVEL
                 ShellCheck minimum severity to fail on. Defaults:
                 local=error, ci=warning
  -h, --help     Show this help text
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for --mode" >&2
        exit 1
      fi
      MODE="$2"
      shift 2
      ;;
    --path)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for --path" >&2
        exit 1
      fi
      PATH_SCOPE="$2"
      shift 2
      ;;
    --shellcheck-severity)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for --shellcheck-severity" >&2
        exit 1
      fi
      SHELLCHECK_SEVERITY="$2"
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

if [[ "$MODE" != "local" && "$MODE" != "ci" ]]; then
  echo "Invalid --mode: $MODE (expected local or ci)" >&2
  exit 1
fi

if [[ "$PATH_SCOPE" != "preflight" && "$PATH_SCOPE" != "vhd" && "$PATH_SCOPE" != "iso" && "$PATH_SCOPE" != "all" ]]; then
  echo "Invalid --path: $PATH_SCOPE (expected preflight, vhd, iso, or all)" >&2
  exit 1
fi

if [[ -z "$SHELLCHECK_SEVERITY" ]]; then
  if [[ "$MODE" == "ci" ]]; then
    SHELLCHECK_SEVERITY="warning"
  else
    SHELLCHECK_SEVERITY="error"
  fi
fi

if [[ "$SHELLCHECK_SEVERITY" != "error" && "$SHELLCHECK_SEVERITY" != "warning" && "$SHELLCHECK_SEVERITY" != "info" && "$SHELLCHECK_SEVERITY" != "style" ]]; then
  echo "Invalid --shellcheck-severity: $SHELLCHECK_SEVERITY (expected error, warning, info, or style)" >&2
  exit 1
fi

failures=0
warnings=0
checks=0

info() {
  printf '[INFO] %s\n' "$1"
}

ok() {
  checks=$((checks + 1))
  printf '[ OK ] %s\n' "$1"
}

warn() {
  warnings=$((warnings + 1))
  printf '[WARN] %s\n' "$1"
}

fail() {
  failures=$((failures + 1))
  printf '[FAIL] %s\n' "$1"
}

check_cmd() {
  local cmd="$1"
  if command -v "$cmd" >/dev/null 2>&1; then
    ok "command present: $cmd"
    return 0
  fi
  return 1
}

shell_files=(
  build.sh
  build-ab-disk.sh
  build-installer-iso.sh
  d2vm
  setup-build-deps.sh
  setup-build-sudoers.sh
  tcfiles/auto-maintenance.debian
  tcfiles/firstboot
  tcfiles/installer-install.sh
  tcfiles/recovery
  tcfiles/set-hostname
  tcfiles/thinclient
)

shopt -s nullglob
for file in tcfiles/tc-*; do
  case "$file" in
    *.service|*.timer)
      continue
      ;;
  esac
  shell_files+=("$file")
done
shopt -u nullglob

strict_mode_files=(
  build.sh
  build-installer-iso.sh
  d2vm
)

exec_files=(
  build.sh
  build-ab-disk.sh
  build-installer-iso.sh
  d2vm
)

required_vhd_cmds=(
  docker
)

required_ab_cmds=(
  sgdisk
  mkfs.fat
  mkfs.ext4
  rsync
  losetup
  grub-install
  grub-editenv
  qemu-img
  blkid
  partprobe
)

required_iso_cmds=(
  xorriso
  qemu-img
  zstd
  curl
  lsblk
  findmnt
)

selected_cmds=()
optional_cmds=()

case "$PATH_SCOPE" in
  preflight)
    selected_cmds=()
    optional_cmds=("${required_vhd_cmds[@]}" "${required_ab_cmds[@]}" "${required_iso_cmds[@]}")
    ;;
  vhd)
    selected_cmds=("${required_vhd_cmds[@]}")
    optional_cmds=("${required_ab_cmds[@]}" "${required_iso_cmds[@]}")
    ;;
  iso)
    selected_cmds=("${required_ab_cmds[@]}" "${required_iso_cmds[@]}")
    optional_cmds=("${required_vhd_cmds[@]}")
    ;;
  all)
    selected_cmds=("${required_vhd_cmds[@]}" "${required_ab_cmds[@]}" "${required_iso_cmds[@]}")
    optional_cmds=()
    ;;
esac

info "Mode=$MODE Path=$PATH_SCOPE ShellCheckSeverity=$SHELLCHECK_SEVERITY"

info "Checking required files exist"
for file in "${shell_files[@]}"; do
  if [[ -f "$file" ]]; then
    ok "found file: $file"
  else
    fail "missing file: $file"
  fi
done

info "Running bash syntax checks"
for file in "${shell_files[@]}"; do
  if [[ ! -f "$file" ]]; then
    continue
  fi
  if bash -n "$file"; then
    ok "bash -n passed: $file"
  else
    fail "bash -n failed: $file"
  fi
done

info "Checking strict mode in build entry scripts"
for file in "${strict_mode_files[@]}"; do
  if [[ ! -f "$file" ]]; then
    continue
  fi
  if grep -Eq '^[[:space:]]*set[[:space:]]+-euo[[:space:]]+pipefail([[:space:]]|$)' "$file"; then
    ok "strict mode present: $file"
  else
    fail "strict mode missing in: $file"
  fi
done

info "Checking help/argument parsing paths"
if bash ./build.sh --help >/dev/null 2>&1; then
  ok "build.sh --help works"
else
  fail "build.sh --help failed"
fi

if bash ./build-installer-iso.sh --help >/dev/null 2>&1; then
  ok "build-installer-iso.sh --help works"
else
  fail "build-installer-iso.sh --help failed"
fi

info "Checking shellcheck availability"
if command -v shellcheck >/dev/null 2>&1; then
  ok "shellcheck present"
  for file in "${shell_files[@]}"; do
    if [[ ! -f "$file" ]]; then
      continue
    fi
    if shellcheck -S "$SHELLCHECK_SEVERITY" "$file"; then
      ok "shellcheck passed: $file"
    else
      fail "shellcheck issues: $file"
    fi
  done
else
  if [[ "$MODE" == "ci" ]]; then
    fail "shellcheck is required in ci mode"
  else
    warn "shellcheck not installed; skipping shell lint"
  fi
fi

info "Checking for CRLF in Linux-consumed scripts"
for file in "${shell_files[@]}"; do
  if [[ ! -f "$file" ]]; then
    continue
  fi
  if grep -q $'\r' "$file"; then
    fail "CRLF detected: $file"
  else
    ok "LF line endings: $file"
  fi
done

info "Checking executable bit on key entrypoints"
for file in "${exec_files[@]}"; do
  if [[ ! -f "$file" ]]; then
    continue
  fi
  if [[ -x "$file" ]]; then
    ok "executable bit set: $file"
  else
    fail "executable bit missing: $file"
  fi
done

info "Checking selected-path required commands"
if [[ ${#selected_cmds[@]} -eq 0 ]]; then
  ok "no selected-path command requirements for path=$PATH_SCOPE"
else
  for cmd in "${selected_cmds[@]}"; do
    if ! check_cmd "$cmd"; then
      fail "required command missing for path=$PATH_SCOPE: $cmd"
    fi
  done
fi

info "Checking non-selected command requirements"
for cmd in "${optional_cmds[@]}"; do
  if command -v "$cmd" >/dev/null 2>&1; then
    ok "optional command present: $cmd"
  else
    warn "optional command missing outside selected path: $cmd"
  fi
done

printf '\nSummary: checks=%d warnings=%d failures=%d\n' "$checks" "$warnings" "$failures"

if [[ "$failures" -gt 0 ]]; then
  exit 1
fi
