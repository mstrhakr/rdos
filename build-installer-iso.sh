#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Build a minimal unattended installer ISO that writes a prebuilt A/B raw disk image to disk and powers off.

CLONEZILLA_VERSION="${CLONEZILLA_VERSION:-3.3.1-35}"
CLONEZILLA_ISO_URL_DEFAULT="https://downloads.sourceforge.net/project/clonezilla/clonezilla_live_stable/${CLONEZILLA_VERSION}/clonezilla-live-${CLONEZILLA_VERSION}-amd64.iso"
CLONEZILLA_ISO_URL="${CLONEZILLA_ISO_URL:-$CLONEZILLA_ISO_URL_DEFAULT}"
CLONEZILLA_ISO_SHA256="${CLONEZILLA_ISO_SHA256:-}"
CLONEZILLA_ALLOW_UNVERIFIED="${CLONEZILLA_ALLOW_UNVERIFIED:-1}"
INPUT_VHD="${INPUT_VHD:-rdos.vhd}"
INPUT_DISK="${INPUT_DISK:-rdos-ab.img}"
INPUT_DISK_ZST="${INPUT_DISK_ZST:-}"
OUTPUT_ISO="${OUTPUT_ISO:-rdos-installer.iso}"
DEFAULT_WORKDIR=".installer-work"
WORKDIR="${WORKDIR:-$DEFAULT_WORKDIR}"
BASE_ISO_CACHE="${BASE_ISO_CACHE:-.installer-cache/clonezilla-base.iso}"
AUTO_INSTALL_DEFAULT="${AUTO_INSTALL_DEFAULT:-0}"
SKIP_BUILD="${SKIP_BUILD:-0}"
BUILD_SCRIPT="${BUILD_SCRIPT:-./build.sh}"
NO_STAGING="${NO_STAGING:-0}"
STAGING_DIR="${STAGING_DIR:-}"
# Compression level for the installer payload. 9 is a good balance (fast + small).
# Set to 19 for maximum compression at the cost of significantly longer build times.
ZSTD_LEVEL="${ZSTD_LEVEL:-9}"

BASE_ISO="$BASE_ISO_CACHE"
ISO_ROOT="$WORKDIR/iso-root"
RAW_IMAGE="$WORKDIR/rdos.img"
COMPRESSED_IMAGE="$ISO_ROOT/RDOS/rdos.img.zst"
IMAGE_SIZE_METADATA="$ISO_ROOT/RDOS/rdos.img.size"
MIN_BASE_ISO_SIZE_BYTES=400000000
TEMP_ROOT="${TMPDIR:-/tmp}"
AUTO_STAGING_DIR=""
ORIGINAL_OUTPUT_ISO="$OUTPUT_ISO"

cleanup_auto_staging() {
  if [[ -n "$AUTO_STAGING_DIR" && -d "$AUTO_STAGING_DIR" ]]; then
    rm -rf "$AUTO_STAGING_DIR"
  fi
}

cleanup_stale_temp_artifacts() {
  if [[ -d "$TEMP_ROOT" ]]; then
    find "$TEMP_ROOT" -maxdepth 1 -mindepth 1 -type d -name 'rdos-iso.*' -exec rm -rf {} + 2>/dev/null || true
  fi
}

trap cleanup_auto_staging EXIT INT TERM HUP

usage() {
  cat <<'EOF'
Usage: ./build-installer-iso.sh [options]

Builds a Clonezilla-based unattended installer ISO for RDOS.
By default, this script builds fresh raw A/B artifacts first by invoking ./build.sh --ab.

Options:
      --skip-build                 Skip invoking build.sh and use existing --input-disk/--input-vhd
      --build-script PATH          Path to build script (default: ./build.sh)
      --build-arg ARG              Pass one argument to build.sh (repeatable)
      --no-cache                   Build without Docker cache (alias for --build-arg --no-cache)
      --input-vhd PATH             Input VHD path (legacy fallback; default: rdos.vhd)
      --input-disk PATH            Input raw disk image (A/B layout; default: rdos-ab.img)
      --input-disk-zst PATH        Input pre-compressed A/B payload (.img.zst)
      --output-iso PATH            Output ISO path (default: rdos-installer.iso)
      --workdir PATH               Working directory for intermediate files
      --staging-dir PATH           Run ISO build work in PATH, then move final ISO to --output-iso
      --no-staging                 Disable automatic /mnt performance staging
      --base-iso-cache PATH        Persistent Clonezilla ISO cache file (default: .installer-cache/clonezilla-base.iso)
      --clonezilla-version VER     Clonezilla release version (default: 3.3.1-35)
      --clonezilla-iso-url URL     Base Clonezilla ISO URL
      --clonezilla-iso-sha256 HEX  Expected SHA256 for the base Clonezilla ISO
      --allow-unverified-downloads Permit unsigned/unchecked base ISO download
      --zstd-level N               Compression level 1-19 (default: 9)
      --auto-install-default       Make unattended installer the default boot entry
  -h, --help                       Show this help message

Environment variables with same names are also supported.
EOF
}

BUILD_SCRIPT_ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-build)
      SKIP_BUILD=1
      shift
      ;;
    --build-script)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      BUILD_SCRIPT="$2"
      shift 2
      ;;
    --build-arg)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      BUILD_SCRIPT_ARGS+=("$2")
      shift 2
      ;;
    --input-vhd)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      INPUT_VHD="$2"
      shift 2
      ;;
      --input-disk)
        if [[ $# -lt 2 ]]; then
          echo "Missing value for $1" >&2
          exit 1
        fi
        INPUT_DISK="$2"
        SKIP_BUILD=1
        shift 2
        ;;
    --input-disk-zst)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      INPUT_DISK_ZST="$2"
      SKIP_BUILD=1
      shift 2
      ;;
    --output-iso)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      OUTPUT_ISO="$2"
      shift 2
      ;;
    --workdir)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      WORKDIR="$2"
      shift 2
      ;;
    --staging-dir)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      STAGING_DIR="$2"
      shift 2
      ;;
    --no-staging)
      NO_STAGING=1
      shift
      ;;
    --no-cache)
      BUILD_SCRIPT_ARGS+=(--no-cache)
      shift
      ;;
    --base-iso-cache)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      BASE_ISO_CACHE="$2"
      BASE_ISO="$BASE_ISO_CACHE"
      shift 2
      ;;
    --clonezilla-iso-url)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      CLONEZILLA_ISO_URL="$2"
      shift 2
      ;;
    --clonezilla-version)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      CLONEZILLA_VERSION="$2"
      CLONEZILLA_ISO_URL="https://downloads.sourceforge.net/project/clonezilla/clonezilla_live_stable/${CLONEZILLA_VERSION}/clonezilla-live-${CLONEZILLA_VERSION}-amd64.iso"
      shift 2
      ;;
    --clonezilla-iso-sha256)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      CLONEZILLA_ISO_SHA256="$2"
      shift 2
      ;;
    --allow-unverified-downloads)
      CLONEZILLA_ALLOW_UNVERIFIED=1
      shift
      ;;
    --zstd-level)
      if [[ $# -lt 2 ]]; then
        echo "Missing value for $1" >&2
        exit 1
      fi
      ZSTD_LEVEL="$2"
      shift 2
      ;;
    --auto-install-default)
      AUTO_INSTALL_DEFAULT=1
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

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

need_cmd xorriso
need_cmd zstd
need_cmd curl
need_cmd lsblk
need_cmd findmnt

log_phase() {
  printf '\n[%s] %s\n' "$(date +'%H:%M:%S')" "$1"
}

copy_with_progress() {
  local source_path="$1"
  local target_path="$2"

  if command -v rsync >/dev/null 2>&1; then
    rsync -ah --info=progress2 --no-inc-recursive --protect-args "$source_path" "$target_path"
  else
    cp -f "$source_path" "$target_path"
  fi
}

verify_clonezilla_iso() {
  local file_path="$1"

  if [[ -n "$CLONEZILLA_ISO_SHA256" ]]; then
    echo "$CLONEZILLA_ISO_SHA256  $file_path" | sha256sum -c -
    return 0
  fi

  if [[ "$CLONEZILLA_ALLOW_UNVERIFIED" == "1" ]]; then
    echo "Warning: base ISO checksum verification is disabled." >&2
    return 0
  fi

  echo "Refusing unverified base ISO download/cache." >&2
  echo "Set CLONEZILLA_ISO_SHA256 or pass --clonezilla-iso-sha256." >&2
  echo "To bypass explicitly, pass --allow-unverified-downloads." >&2
  return 1
}

if [[ ! "$ZSTD_LEVEL" =~ ^[0-9]+$ ]] || (( ZSTD_LEVEL < 1 || ZSTD_LEVEL > 19 )); then
  echo "Invalid --zstd-level: $ZSTD_LEVEL (expected 1-19)" >&2
  exit 1
fi

if [[ -z "$STAGING_DIR" ]] && [[ "$NO_STAGING" != "1" ]] && [[ "$SCRIPT_DIR" == /mnt/* ]] && [[ "$WORKDIR" == "$DEFAULT_WORKDIR" ]]; then
  cleanup_stale_temp_artifacts
  AUTO_STAGING_DIR="$(mktemp -d "$TEMP_ROOT/rdos-iso.XXXXXX")"
  STAGING_DIR="$AUTO_STAGING_DIR"
fi

if [[ -n "$STAGING_DIR" ]]; then
  WORKDIR="$STAGING_DIR/work"
  OUTPUT_ISO="$STAGING_DIR/$(basename "$ORIGINAL_OUTPUT_ISO")"
  log_phase "Detected /mnt workspace. Using fast staging at $STAGING_DIR for ISO build"
fi

ISO_ROOT="$WORKDIR/iso-root"
RAW_IMAGE="$WORKDIR/rdos.img"
COMPRESSED_IMAGE="$ISO_ROOT/RDOS/rdos.img.zst"
IMAGE_SIZE_METADATA="$ISO_ROOT/RDOS/rdos.img.size"

if [[ "$SKIP_BUILD" != "1" ]]; then
  if [[ ! -x "$BUILD_SCRIPT" ]]; then
    echo "Build script not found or not executable: $BUILD_SCRIPT" >&2
    exit 1
  fi

  build_output_disk="$INPUT_DISK"
  if [[ -n "$STAGING_DIR" ]]; then
    build_output_disk="$STAGING_DIR/$(basename "$INPUT_DISK")"
  fi

  log_phase "Building fresh raw A/B disk via $BUILD_SCRIPT"
  "$BUILD_SCRIPT" --ab --output-ab "$build_output_disk" --output-ab-zst "${build_output_disk}.zst" --force "${BUILD_SCRIPT_ARGS[@]}"
  INPUT_DISK="$build_output_disk"
  INPUT_DISK_ZST="${build_output_disk}.zst"
else
  if [[ -n "$INPUT_DISK_ZST" ]]; then
    log_phase "Skipping build; using existing compressed disk payload $INPUT_DISK_ZST"
  elif [[ -n "$INPUT_DISK" ]]; then
    log_phase "Skipping build; using existing raw disk $INPUT_DISK"
  else
    log_phase "Skipping VHD build; using existing $INPUT_VHD"
  fi
fi

if [[ -n "$INPUT_DISK_ZST" ]]; then
  if [[ ! -f "$INPUT_DISK_ZST" ]]; then
    echo "Input compressed disk not found: $INPUT_DISK_ZST" >&2
    exit 1
  fi
elif [[ -n "$INPUT_DISK" ]]; then
  if [[ ! -f "$INPUT_DISK" ]]; then
    echo "Input disk not found: $INPUT_DISK" >&2
    exit 1
  fi
else
  need_cmd qemu-img

  if [[ ! -f "$INPUT_VHD" ]]; then
    echo "Input VHD not found: $INPUT_VHD" >&2
    echo "Re-run without --skip-build, or set --input-vhd to an existing file." >&2
    exit 1
  fi
fi

if [[ -z "$INPUT_DISK" ]] && [[ -n "$STAGING_DIR" ]] && [[ "$INPUT_VHD" != "$STAGING_DIR/$(basename "$INPUT_VHD")" ]]; then
  staged_input_vhd="$STAGING_DIR/$(basename "$INPUT_VHD")"
  log_phase "Copying input VHD to staging area"
  copy_with_progress "$INPUT_VHD" "$staged_input_vhd"
  INPUT_VHD="$staged_input_vhd"
fi

if [[ -n "$INPUT_DISK" ]] && [[ -n "$STAGING_DIR" ]] && [[ "$INPUT_DISK" != "$STAGING_DIR/$(basename "$INPUT_DISK")" ]]; then
  staged_input_disk="$STAGING_DIR/$(basename "$INPUT_DISK")"
  log_phase "Copying input disk to staging area"
  copy_with_progress "$INPUT_DISK" "$staged_input_disk"
  INPUT_DISK="$staged_input_disk"
fi

if [[ -n "$INPUT_DISK_ZST" ]] && [[ -n "$STAGING_DIR" ]] && [[ "$INPUT_DISK_ZST" != "$STAGING_DIR/$(basename "$INPUT_DISK_ZST")" ]]; then
  staged_input_disk_zst="$STAGING_DIR/$(basename "$INPUT_DISK_ZST")"
  log_phase "Copying compressed disk payload to staging area"
  copy_with_progress "$INPUT_DISK_ZST" "$staged_input_disk_zst"
  INPUT_DISK_ZST="$staged_input_disk_zst"
fi

mkdir -p "$WORKDIR"
mkdir -p "$(dirname "$BASE_ISO")"

need_download=0
if [[ ! -f "$BASE_ISO" ]]; then
  need_download=1
else
  base_size="$(wc -c <"$BASE_ISO" || echo 0)"
  if [[ "$base_size" -lt "$MIN_BASE_ISO_SIZE_BYTES" ]]; then
    echo "Cached base ISO looks too small ($base_size bytes). Re-downloading..."
    rm -f "$BASE_ISO"
    need_download=1
  fi
fi

if [[ "$need_download" == "1" ]]; then
  log_phase "Downloading Clonezilla Live base ISO"
  curl -L "$CLONEZILLA_ISO_URL" -o "$BASE_ISO"
fi

verify_clonezilla_iso "$BASE_ISO"

# Quick integrity sanity check: ensure the largest live payload exists in the ISO.
if ! xorriso -indev "$BASE_ISO" -find /live/filesystem.squashfs -exec report_lba >/dev/null 2>&1; then
  log_phase "Cached base ISO appears corrupted or incomplete; re-downloading"
  rm -f "$BASE_ISO"
  curl -L "$CLONEZILLA_ISO_URL" -o "$BASE_ISO"

  if ! xorriso -indev "$BASE_ISO" -find /live/filesystem.squashfs -exec report_lba >/dev/null 2>&1; then
    echo "Downloaded base ISO still failed integrity sanity check." >&2
    exit 1
  fi
fi

log_phase "Extracting base ISO"
rm -rf "$ISO_ROOT"
mkdir -p "$ISO_ROOT"
xorriso -osirrox on -indev "$BASE_ISO" -extract / "$ISO_ROOT" >/dev/null
# Extracted ISO files can be read-only; make them writable before patching configs.
chmod -R u+w "$ISO_ROOT"

# Clonezilla's GRUB config includes a startup "play" tone. Remove it to mute menu beeps.
for grub_file in "$ISO_ROOT/boot/grub/grub.cfg" "$ISO_ROOT/boot/grub/config.cfg"; do
  if [[ -f "$grub_file" ]]; then
    sed -i '/^[[:space:]]*play[[:space:]]\+/d' "$grub_file"
  fi
done

mkdir -p "$ISO_ROOT/RDOS"

if [[ -n "$INPUT_DISK_ZST" ]]; then
  log_phase "Using pre-compressed A/B payload directly: $INPUT_DISK_ZST"
  cp "$INPUT_DISK_ZST" "$COMPRESSED_IMAGE"

  if [[ -f "${INPUT_DISK_ZST}.size" ]]; then
    cp "${INPUT_DISK_ZST}.size" "$IMAGE_SIZE_METADATA"
  elif [[ -n "$INPUT_DISK" && -f "$INPUT_DISK" ]]; then
    stat -c%s "$INPUT_DISK" > "$IMAGE_SIZE_METADATA"
  else
    log_phase "No .size metadata found; deriving uncompressed size from payload stream"
    zstd -d -c "$INPUT_DISK_ZST" | wc -c > "$IMAGE_SIZE_METADATA"
  fi
elif [[ -n "$INPUT_DISK" ]]; then
  log_phase "Using raw A/B disk image directly: $INPUT_DISK"
  cp "$INPUT_DISK" "$RAW_IMAGE"
  log_phase "Compressing installer payload (zstd level ${ZSTD_LEVEL}, threads: all)"
  zstd -T0 "-${ZSTD_LEVEL}" -f "$RAW_IMAGE" -o "$COMPRESSED_IMAGE"
  stat -c%s "$RAW_IMAGE" > "$IMAGE_SIZE_METADATA"
  rm -f "$RAW_IMAGE"
else
  log_phase "Converting $INPUT_VHD to raw image"
  qemu-img convert -f vpc -O raw "$INPUT_VHD" "$RAW_IMAGE"
  log_phase "Compressing installer payload (zstd level ${ZSTD_LEVEL}, threads: all)"
  zstd -T0 "-${ZSTD_LEVEL}" -f "$RAW_IMAGE" -o "$COMPRESSED_IMAGE"
  stat -c%s "$RAW_IMAGE" > "$IMAGE_SIZE_METADATA"
  rm -f "$RAW_IMAGE"
fi

cp "$SCRIPT_DIR/tcfiles/installer-install.sh" "$ISO_ROOT/RDOS/install.sh"
cp "$SCRIPT_DIR/tcfiles/tc-installer-ui.sh" "$ISO_ROOT/RDOS/tc-installer-ui.sh"
chmod +x "$ISO_ROOT/RDOS/install.sh"
chmod +x "$ISO_ROOT/RDOS/tc-installer-ui.sh"

CLONEZILLA_BOOT_ARGS="boot=live union=overlay username=user config components quiet loglevel=3 noswap edd=on nomodeset enforcing=0 locales= keyboard-layouts= net.ifnames=0 nosplash modprobe.blacklist=pcspkr"

for SYS_CFG in "$ISO_ROOT/syslinux/syslinux.cfg" "$ISO_ROOT/syslinux/isolinux.cfg"; do
  if [[ -f "$SYS_CFG" ]]; then
    log_phase "Writing BIOS boot config in $(basename "$SYS_CFG")"
    cat >"$SYS_CFG" <<EOF
default RDOS_guided
prompt 0
timeout 50

menu title RDOS Installer

label RDOS_guided
  menu label RDOS guided installer (disk selection + progress UI)
  kernel /live/vmlinuz
  append initrd=/live/initrd.img ${CLONEZILLA_BOOT_ARGS} ocs_live_run="bash /run/live/medium/RDOS/install.sh guided" ocs_live_extra_param="" ocs_live_batch="yes"

label RDOS_auto
  menu label RDOS automatic install (first target disk)
  kernel /live/vmlinuz
  append initrd=/live/initrd.img ${CLONEZILLA_BOOT_ARGS} ocs_live_run="bash /run/live/medium/RDOS/install.sh auto" ocs_live_extra_param="" ocs_live_batch="yes"
EOF
  fi
done

GRUB_CFG="$ISO_ROOT/boot/grub/grub.cfg"
if [[ -f "$GRUB_CFG" ]]; then
  log_phase "Writing GRUB boot config"
  cat >"$GRUB_CFG" <<EOF
set default="0"
set timeout=5
set timeout_style=menu

menuentry "RDOS guided installer (disk selection + progress UI)" --id RDOS_guided {
  linux /live/vmlinuz ${CLONEZILLA_BOOT_ARGS} ocs_live_run="bash /run/live/medium/RDOS/install.sh guided" ocs_live_extra_param="" ocs_live_batch="yes"
  initrd /live/initrd.img
}

menuentry "RDOS automatic install (first target disk)" --id RDOS_auto {
  linux /live/vmlinuz ${CLONEZILLA_BOOT_ARGS} ocs_live_run="bash /run/live/medium/RDOS/install.sh auto" ocs_live_extra_param="" ocs_live_batch="yes"
  initrd /live/initrd.img
}
EOF
fi

ISOLINUX_BIN="syslinux/isolinux.bin"
BOOT_CAT="syslinux/boot.cat"
EFI_BOOT_IMG=""

if [[ ! -f "$ISO_ROOT/$ISOLINUX_BIN" ]]; then
  echo "Expected BIOS boot file not found: $ISOLINUX_BIN" >&2
  exit 1
fi

for candidate in \
  "$ISO_ROOT/boot/grub/efi.img" \
  "$ISO_ROOT/boot/grub/EFI.img" \
  "$ISO_ROOT/EFI/boot/efi.img" \
  "$ISO_ROOT/EFI/BOOT/EFI.img"; do
  if [[ -f "$candidate" ]]; then
    EFI_BOOT_IMG="${candidate#"$ISO_ROOT"/}"
    break
  fi
done

if [[ -z "$EFI_BOOT_IMG" ]]; then
  echo "Expected UEFI boot image efi.img was not found in extracted ISO." >&2
  exit 1
fi

log_phase "Building installer ISO"
xorriso -as mkisofs \
  -r -J -joliet-long -l -iso-level 3 \
  -V "RDOS_INSTALLER" \
  -o "$OUTPUT_ISO" \
  -b "$ISOLINUX_BIN" \
  -c "$BOOT_CAT" \
  -no-emul-boot \
  -boot-load-size 4 \
  -boot-info-table \
  -eltorito-alt-boot \
  -e "$EFI_BOOT_IMG" \
  -no-emul-boot \
  "$ISO_ROOT"

log_phase "Installer ISO build complete"
echo "Created $OUTPUT_ISO"
echo "Boot mode: guided installer default (5s timeout) with optional automatic mode."
echo "Payload mode: zstd compressed."
echo "Review target disk detection in RDOS/install.sh if you need a different policy."

if [[ -n "$STAGING_DIR" ]]; then
  log_phase "Publishing staged ISO to destination: $ORIGINAL_OUTPUT_ISO"
  if [[ -e "$ORIGINAL_OUTPUT_ISO" ]]; then
    chmod u+w "$ORIGINAL_OUTPUT_ISO" 2>/dev/null || true
    rm -f "$ORIGINAL_OUTPUT_ISO"
  fi
  copy_with_progress "$OUTPUT_ISO" "$ORIGINAL_OUTPUT_ISO"
  echo "Created $ORIGINAL_OUTPUT_ISO"
fi
