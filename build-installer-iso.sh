#!/usr/bin/env bash
set -euo pipefail

# Build a minimal unattended installer ISO that writes uftc.vhd to disk and powers off.

CLONEZILLA_ISO_URL="${CLONEZILLA_ISO_URL:-https://downloads.sourceforge.net/project/clonezilla/clonezilla_live_stable/3.2.2-5/clonezilla-live-3.2.2-5-amd64.iso}"
INPUT_VHD="${INPUT_VHD:-uftc.vhd}"
OUTPUT_ISO="${OUTPUT_ISO:-uftc-installer.iso}"
WORKDIR="${WORKDIR:-.installer-work}"

BASE_ISO="$WORKDIR/clonezilla-base.iso"
ISO_ROOT="$WORKDIR/iso-root"
RAW_IMAGE="$WORKDIR/uftc.img"
COMPRESSED_IMAGE="$ISO_ROOT/uftc/uftc.img.zst"
MIN_BASE_ISO_SIZE_BYTES=400000000

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

need_cmd xorriso
need_cmd qemu-img
need_cmd zstd
need_cmd curl
need_cmd lsblk
need_cmd findmnt

log_phase() {
  printf '\n[%s] %s\n' "$(date +'%H:%M:%S')" "$1"
}

if [[ ! -f "$INPUT_VHD" ]]; then
  echo "Input VHD not found: $INPUT_VHD" >&2
  echo "Run ./build.sh first or set INPUT_VHD=/path/to/uftc.vhd" >&2
  exit 1
fi

mkdir -p "$WORKDIR"

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

mkdir -p "$ISO_ROOT/uftc"

log_phase "Converting $INPUT_VHD to raw image"
qemu-img convert -f vpc -O raw "$INPUT_VHD" "$RAW_IMAGE"

log_phase "Compressing installer payload"
zstd -T0 -19 -f "$RAW_IMAGE" -o "$COMPRESSED_IMAGE"
rm -f "$RAW_IMAGE"

cat >"$ISO_ROOT/uftc/install.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

LOG_FILE="/var/log/uftc-installer.log"
touch "$LOG_FILE"
exec > >(tee -a "$LOG_FILE") 2>&1

echo "=== UFTC unattended installer ==="
date

IMAGE_ZST="/run/live/medium/uftc/uftc.img.zst"
if [[ ! -f "$IMAGE_ZST" ]]; then
  IMAGE_ZST="/lib/live/mount/medium/uftc/uftc.img.zst"
fi

if [[ ! -f "$IMAGE_ZST" ]]; then
  echo "Installer image payload missing from live media."
  poweroff -f
fi

LIVE_SOURCE="$(findmnt -n -o SOURCE /run/live/medium 2>/dev/null || true)"
if [[ -z "$LIVE_SOURCE" ]]; then
  LIVE_SOURCE="$(findmnt -n -o SOURCE /lib/live/mount/medium 2>/dev/null || true)"
fi
LIVE_DISK=""

if [[ -n "$LIVE_SOURCE" ]]; then
  LIVE_RESOLVED="$(readlink -f "$LIVE_SOURCE" || echo "$LIVE_SOURCE")"
  LIVE_DISK="$(lsblk -ndo PKNAME "$LIVE_RESOLVED" 2>/dev/null || true)"
fi

TARGET_DISK=""

while read -r disk _type; do
  if [[ -z "$disk" ]]; then
    continue
  fi

  if [[ "$disk" == "$LIVE_DISK" ]]; then
    continue
  fi

  if [[ -f "/sys/block/$disk/removable" ]] && [[ "$(cat "/sys/block/$disk/removable")" != "0" ]]; then
    continue
  fi

  TARGET_DISK="$disk"
  break
done < <(lsblk -ndo NAME,TYPE | awk '$2=="disk" { print $1, $2 }')

if [[ -z "$TARGET_DISK" ]]; then
  echo "No install target disk was found."
  echo "Live media disk was: ${LIVE_DISK:-unknown}"
  poweroff -f
fi

echo "Writing image to /dev/$TARGET_DISK"
echo "This will erase all data on /dev/$TARGET_DISK"

zstd -d -c "$IMAGE_ZST" | dd of="/dev/$TARGET_DISK" bs=16M status=progress conv=fsync
sync

echo "Install complete. Powering off."
poweroff -f
EOF

chmod +x "$ISO_ROOT/uftc/install.sh"

CLONEZILLA_BOOT_ARGS="boot=live union=overlay username=user config components quiet loglevel=3 ocs_1_cpu_udev noswap edd=on nomodeset enforcing=0 locales= keyboard-layouts= net.ifnames=0 nosplash"

for SYS_CFG in "$ISO_ROOT/syslinux/syslinux.cfg" "$ISO_ROOT/syslinux/isolinux.cfg"; do
  if [[ -f "$SYS_CFG" ]] && ! grep -q "label uftc_auto" "$SYS_CFG"; then
    log_phase "Patching BIOS boot menu in $(basename "$SYS_CFG")"
    cat >>"$SYS_CFG" <<EOF

label uftc_auto
  menu label UFTC automatic install (erase target disk)
  kernel /live/vmlinuz
  append initrd=/live/initrd.img ${CLONEZILLA_BOOT_ARGS} ocs_live_run="bash /run/live/medium/uftc/install.sh" ocs_live_extra_param="" ocs_live_batch="yes"
EOF

    if grep -q '^default ' "$SYS_CFG"; then
      sed -i 's/^default .*/default uftc_auto/' "$SYS_CFG"
    else
      sed -i '1idefault uftc_auto' "$SYS_CFG"
    fi
  fi
done

GRUB_CFG="$ISO_ROOT/boot/grub/grub.cfg"
if [[ -f "$GRUB_CFG" ]] && ! grep -q -- "--id uftc_auto" "$GRUB_CFG"; then
  log_phase "Patching GRUB boot menu"
  cat >>"$GRUB_CFG" <<EOF

menuentry "UFTC automatic install (erase target disk)" --id uftc_auto {
  linux /live/vmlinuz ${CLONEZILLA_BOOT_ARGS} ocs_live_run="bash /run/live/medium/uftc/install.sh" ocs_live_extra_param="" ocs_live_batch="yes"
  initrd /live/initrd.img
}
EOF

  if grep -q '^set default=' "$GRUB_CFG"; then
    sed -i 's/^set default=.*/set default="uftc_auto"/' "$GRUB_CFG"
  else
    sed -i '1iset default="uftc_auto"' "$GRUB_CFG"
  fi
fi

ISOLINUX_BIN="syslinux/isolinux.bin"
BOOT_CAT="syslinux/boot.cat"

if [[ ! -f "$ISO_ROOT/$ISOLINUX_BIN" ]]; then
  echo "Expected BIOS boot file not found: $ISOLINUX_BIN" >&2
  exit 1
fi

EFI_BOOT_REL=""
while IFS= read -r efi_file; do
  EFI_BOOT_REL="${efi_file#$ISO_ROOT/}"
  break
done < <(find "$ISO_ROOT" -type f \( -iname 'bootx64.efi' -o -iname 'BOOTx64.EFI' \))

if [[ -z "$EFI_BOOT_REL" ]]; then
  echo "Expected UEFI boot file BOOTx64.EFI was not found in extracted ISO." >&2
  exit 1
fi

log_phase "Building installer ISO"
xorriso -as mkisofs \
  -r -J -joliet-long -l -iso-level 3 \
  -V "UFTC_INSTALLER" \
  -o "$OUTPUT_ISO" \
  -b "$ISOLINUX_BIN" \
  -c "$BOOT_CAT" \
  -no-emul-boot \
  -boot-load-size 4 \
  -boot-info-table \
  -eltorito-alt-boot \
  -e "$EFI_BOOT_REL" \
  -no-emul-boot \
  "$ISO_ROOT"

log_phase "Installer ISO build complete"
echo "Created $OUTPUT_ISO"
echo "This ISO starts the installer automatically, writes UFTC to the first non-removable disk, then powers off."
echo "Payload mode: zstd compressed."
echo "Review target disk detection in uftc/install.sh if you need a different policy."
