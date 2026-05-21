#!/usr/bin/env bash
# build-ab-disk.sh — Assemble a 31 GB A/B + Recovery disk from the production
# and recovery VHDs produced by d2vm.
#
# Partition layout (GPT):
#   p1  1 MB    BIOSBOOT  (type EF02, no filesystem — GRUB BIOS core)
#   p2  4 GB    ESP       (FAT32, label ESP — GRUB modules, grub.cfg, grubenv,
#                          slot kernels: vmlinuz-a/b, initrd-a/b.img)
#   p3  12 GB   ROOT_A    (ext4, label ROOT_A — initial production OS)
#   p4  12 GB   ROOT_B    (ext4, label ROOT_B — empty, filled by OTA)
#   p5   2 GB   RECOVERY  (ext4, label RECOVERY — Alpine recovery OS)
#
# Usage:
#   sudo ./build-ab-disk.sh [OPTIONS]
#
# Options:
#   --prod-vhd PATH       Production VHD (default: uftc.vhd)
#   --recovery-vhd PATH   Recovery VHD   (default: recovery.vhd)
#   --prod-raw PATH       Production raw disk image (optional; skips VHD conversion)
#   --recovery-raw PATH   Recovery raw disk image   (optional; skips VHD conversion)
#   --output PATH         Output disk image (default: uftc-ab.img)
#   --skip-efi            Skip EFI GRUB install (BIOS only)
#
set -euo pipefail

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
PROD_VHD="uftc.vhd"
RECOVERY_VHD="recovery.vhd"
PROD_RAW_INPUT=""
RECOVERY_RAW_INPUT=""
OUTPUT_DISK="uftc-ab.img"
SKIP_EFI=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --prod-vhd)       PROD_VHD="$2";     shift 2 ;;
        --recovery-vhd)   RECOVERY_VHD="$2"; shift 2 ;;
        --prod-raw)       PROD_RAW_INPUT="$2"; shift 2 ;;
        --recovery-raw)   RECOVERY_RAW_INPUT="$2"; shift 2 ;;
        --output)         OUTPUT_DISK="$2";  shift 2 ;;
        --skip-efi)       SKIP_EFI=true;     shift   ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

log() { echo "[build-ab-disk] $*"; }
die() { echo "[build-ab-disk] FATAL: $*" >&2; exit 1; }

PARTITION_TOOL="partscan"

wait_for_block_device() {
    local dev_path="$1"
    local timeout_sec="${2:-30}"
    local i

    for ((i=0; i<timeout_sec; i++)); do
        if [[ -b "$dev_path" ]]; then
            return 0
        fi
        if command -v udevadm >/dev/null 2>&1; then
            udevadm settle 2>/dev/null || true
        fi
        sleep 1
    done

    return 1
}

prepare_loop_partitions() {
    local loop_dev="$1"
    local mapper_probe
    mapper_probe="/dev/mapper/$(basename "$loop_dev")p1"

    if command -v kpartx >/dev/null 2>&1; then
        if kpartx -as "$loop_dev" >/dev/null 2>&1 || kpartx -a "$loop_dev" >/dev/null 2>&1; then
            if command -v udevadm >/dev/null 2>&1; then
                udevadm settle 2>/dev/null || true
            fi

            # Some WSL setups have kpartx installed but do not expose /dev/mapper nodes.
            if [[ -b "$mapper_probe" ]]; then
                PARTITION_TOOL="kpartx"
                return 0
            fi

            kpartx -d "$loop_dev" >/dev/null 2>&1 || true
        fi
    fi

    PARTITION_TOOL="partscan"
    partprobe "$loop_dev" 2>/dev/null || true
    if command -v partx >/dev/null 2>&1; then
        partx -a "$loop_dev" 2>/dev/null || partx -u "$loop_dev" 2>/dev/null || true
    fi
    if command -v udevadm >/dev/null 2>&1; then
        udevadm settle 2>/dev/null || true
    fi
}

partition_device() {
    local loop_dev="$1"
    local part_num="$2"

    if [[ "$PARTITION_TOOL" == "kpartx" ]]; then
        printf '/dev/mapper/%s' "$(basename "$loop_dev")p${part_num}"
    else
        printf '%sp%s' "$loop_dev" "$part_num"
    fi
}

# ---------------------------------------------------------------------------
# Dependency check
# ---------------------------------------------------------------------------
for cmd in sgdisk mkfs.fat mkfs.ext4 rsync losetup grub-install grub-editenv \
           qemu-img blkid partprobe; do
    command -v "$cmd" >/dev/null 2>&1 || die "Required tool not found: $cmd"
done

if [[ -n "$PROD_RAW_INPUT" ]]; then
    [[ -f "$PROD_RAW_INPUT" ]] || die "Production raw image not found: $PROD_RAW_INPUT"
else
    [[ -f "$PROD_VHD" ]] || die "Production VHD not found: $PROD_VHD"
fi

if [[ -n "$RECOVERY_RAW_INPUT" ]]; then
    [[ -f "$RECOVERY_RAW_INPUT" ]] || die "Recovery raw image not found: $RECOVERY_RAW_INPUT"
else
    [[ -f "$RECOVERY_VHD" ]] || die "Recovery VHD not found: $RECOVERY_VHD"
fi

# ---------------------------------------------------------------------------
# Temp directory and cleanup
# ---------------------------------------------------------------------------
WORK_DIR="$(mktemp -d /var/tmp/uftc-ab.XXXXXX)"

AB_LOOP=""
VHD_LOOP=""
REC_LOOP=""

cleanup() {
    local rc=$?
    set +e
    for mp in root_a/sys root_a/proc root_a/dev root_a/boot \
               root_a root_b recovery esp vhd_boot vhd_root rec_boot rec_root; do
        umount -l "$WORK_DIR/$mp" 2>/dev/null || true
    done
    if [[ "$PARTITION_TOOL" == "kpartx" ]]; then
        [[ -n "$AB_LOOP"  ]] && kpartx -d "$AB_LOOP"  >/dev/null 2>&1 || true
        [[ -n "$VHD_LOOP" ]] && kpartx -d "$VHD_LOOP" >/dev/null 2>&1 || true
        [[ -n "$REC_LOOP" ]] && kpartx -d "$REC_LOOP" >/dev/null 2>&1 || true
    fi
    [[ -n "$AB_LOOP"  ]] && losetup -d "$AB_LOOP"  2>/dev/null || true
    [[ -n "$VHD_LOOP" ]] && losetup -d "$VHD_LOOP" 2>/dev/null || true
    [[ -n "$REC_LOOP" ]] && losetup -d "$REC_LOOP" 2>/dev/null || true
    rm -rf "$WORK_DIR"
    [[ $rc -eq 0 ]] || echo "[build-ab-disk] Build failed (exit $rc)"
}
trap cleanup EXIT

mkdir -p "$WORK_DIR"/{esp,root_a,root_b,recovery,vhd_boot,vhd_root,rec_boot,rec_root}

# ---------------------------------------------------------------------------
# Create and partition the output disk
# ---------------------------------------------------------------------------
log "Creating output disk: $OUTPUT_DISK"
rm -f "$OUTPUT_DISK"
truncate -s 31G "$OUTPUT_DISK"

log "Partitioning with GPT"
# Clear any existing partition table
sgdisk --zap-all "$OUTPUT_DISK"

# Create all partitions in one write to avoid repeated "old partition table" warnings.
sgdisk \
    -n 1:2048:+1M  -t 1:ef02 -c 1:BIOSBOOT \
    -n 2:0:+4G     -t 2:ef00 -c 2:ESP \
    -n 3:0:+12G    -t 3:8300 -c 3:ROOT_A \
    -n 4:0:+12G    -t 4:8300 -c 4:ROOT_B \
    -n 5:0:0       -t 5:8300 -c 5:RECOVERY \
    "$OUTPUT_DISK"

AB_LOOP=$(losetup --find --show --partscan "$OUTPUT_DISK")
prepare_loop_partitions "$AB_LOOP"
for part in 2 3 4 5; do
    part_dev="$(partition_device "$AB_LOOP" "$part")"
    wait_for_block_device "$part_dev" 30 || die "Partition node $part_dev did not appear in time"
done

log "Formatting partitions"
mkfs.fat -F32 -n ESP      "$(partition_device "$AB_LOOP" 2)"
mkfs.ext4 -q -L ROOT_A    "$(partition_device "$AB_LOOP" 3)"
mkfs.ext4 -q -L ROOT_B    "$(partition_device "$AB_LOOP" 4)"
mkfs.ext4 -q -L RECOVERY  "$(partition_device "$AB_LOOP" 5)"

mount "$(partition_device "$AB_LOOP" 2)" "$WORK_DIR/esp"
mount "$(partition_device "$AB_LOOP" 3)" "$WORK_DIR/root_a"
mount "$(partition_device "$AB_LOOP" 5)" "$WORK_DIR/recovery"

# ---------------------------------------------------------------------------
# Extract production VHD (d2vm output)
# d2vm layout: p1 = FAT32 boot (contains kernel + GRUB), p2 = ext4 root
# ---------------------------------------------------------------------------
log "Preparing production source image"
PROD_RAW="$PROD_RAW_INPUT"
if [[ -z "$PROD_RAW" ]]; then
    PROD_RAW="$WORK_DIR/prod.raw"
    log "Converting production VHD to raw image for reliable loop partition probing"
    qemu-img convert -f vpc -O raw "$PROD_VHD" "$PROD_RAW"
else
    log "Using prebuilt production raw image: $PROD_RAW"
fi

VHD_LOOP=$(losetup --find --show --partscan "$PROD_RAW")
prepare_loop_partitions "$VHD_LOOP"
part_dev="$(partition_device "$VHD_LOOP" 1)"
wait_for_block_device "$part_dev" 30 || die "Partition node $part_dev did not appear in time"
part_dev="$(partition_device "$VHD_LOOP" 2)"
wait_for_block_device "$part_dev" 30 || die "Partition node $part_dev did not appear in time"

mount -o ro "$(partition_device "$VHD_LOOP" 1)" "$WORK_DIR/vhd_boot"
mount -o ro "$(partition_device "$VHD_LOOP" 2)" "$WORK_DIR/vhd_root"

log "Copying production root filesystem into ROOT_A"
rsync -aAX \
    --exclude='/proc/*' \
    --exclude='/sys/*' \
    --exclude='/dev/*' \
    --exclude='/run/*' \
    --exclude='/tmp/*' \
    "$WORK_DIR/vhd_root/" "$WORK_DIR/root_a/"

log "Copying production kernel to ESP as vmlinuz-a / initrd-a.img"
# d2vm places the kernel in the FAT32 boot partition; find it by name pattern.
PROD_VMLINUZ=$(find "$WORK_DIR/vhd_boot" -maxdepth 1 -name 'vmlinuz*' | sort -V | tail -1)
PROD_INITRD=$(find  "$WORK_DIR/vhd_boot" -maxdepth 1 -name 'initrd*'  | sort -V | tail -1)
# Fall back to plain symlinks if versioned names aren't present
[ -z "$PROD_VMLINUZ" ] && PROD_VMLINUZ="$WORK_DIR/vhd_boot/vmlinuz"
[ -z "$PROD_INITRD"  ] && PROD_INITRD="$WORK_DIR/vhd_boot/initrd.img"
[[ -f "$PROD_VMLINUZ" ]] || die "Kernel not found in production VHD boot partition"
[[ -f "$PROD_INITRD"  ]] || die "Initrd not found in production VHD boot partition"
cp "$PROD_VMLINUZ" "$WORK_DIR/esp/vmlinuz-a"
cp "$PROD_INITRD"  "$WORK_DIR/esp/initrd-a.img"

umount "$WORK_DIR/vhd_root"
umount "$WORK_DIR/vhd_boot"
if [[ "$PARTITION_TOOL" == "kpartx" ]]; then
    kpartx -d "$VHD_LOOP" >/dev/null 2>&1 || true
fi
losetup -d "$VHD_LOOP"; VHD_LOOP=""

# ---------------------------------------------------------------------------
# Extract recovery VHD
# d2vm layout (same): p1 = FAT32 boot (Alpine kernel), p2 = ext4 Alpine root
# ---------------------------------------------------------------------------
log "Preparing recovery source image"
RECOVERY_RAW="$RECOVERY_RAW_INPUT"
if [[ -z "$RECOVERY_RAW" ]]; then
    RECOVERY_RAW="$WORK_DIR/recovery.raw"
    log "Converting recovery VHD to raw image for reliable loop partition probing"
    qemu-img convert -f vpc -O raw "$RECOVERY_VHD" "$RECOVERY_RAW"
else
    log "Using prebuilt recovery raw image: $RECOVERY_RAW"
fi

REC_LOOP=$(losetup --find --show --partscan "$RECOVERY_RAW")
prepare_loop_partitions "$REC_LOOP"
part_dev="$(partition_device "$REC_LOOP" 1)"
wait_for_block_device "$part_dev" 30 || die "Partition node $part_dev did not appear in time"
part_dev="$(partition_device "$REC_LOOP" 2)"
wait_for_block_device "$part_dev" 30 || die "Partition node $part_dev did not appear in time"

mount -o ro "$(partition_device "$REC_LOOP" 1)" "$WORK_DIR/rec_boot"
mount -o ro "$(partition_device "$REC_LOOP" 2)" "$WORK_DIR/rec_root"

log "Copying recovery root filesystem into RECOVERY"
rsync -aAX \
    --exclude='/proc/*' \
    --exclude='/sys/*' \
    --exclude='/dev/*' \
    --exclude='/run/*' \
    --exclude='/tmp/*' \
    "$WORK_DIR/rec_root/" "$WORK_DIR/recovery/"

log "Copying recovery kernel to RECOVERY/boot"
mkdir -p "$WORK_DIR/recovery/boot"
REC_VMLINUZ=$(find "$WORK_DIR/rec_boot" -maxdepth 1 -name 'vmlinuz*' | sort -V | tail -1)
REC_INITRD=$(find  "$WORK_DIR/rec_boot" -maxdepth 1 -name 'initrd*'  | sort -V | tail -1)
[ -z "$REC_VMLINUZ" ] && REC_VMLINUZ="$WORK_DIR/rec_boot/vmlinuz"
[ -z "$REC_INITRD"  ] && REC_INITRD="$WORK_DIR/rec_boot/initrd.img"
[[ -f "$REC_VMLINUZ" ]] || die "Kernel not found in recovery VHD boot partition"
[[ -f "$REC_INITRD"  ]] || die "Initrd not found in recovery VHD boot partition"
cp "$REC_VMLINUZ" "$WORK_DIR/recovery/boot/vmlinuz"
cp "$REC_INITRD"  "$WORK_DIR/recovery/boot/initrd.img"

umount "$WORK_DIR/rec_root"
umount "$WORK_DIR/rec_boot"
if [[ "$PARTITION_TOOL" == "kpartx" ]]; then
    kpartx -d "$REC_LOOP" >/dev/null 2>&1 || true
fi
losetup -d "$REC_LOOP"; REC_LOOP=""

# ---------------------------------------------------------------------------
# Update /etc/fstab in ROOT_A to reference partitions by label
# ---------------------------------------------------------------------------
log "Writing /etc/fstab in ROOT_A"
cat > "$WORK_DIR/root_a/etc/fstab" <<'FSTAB'
# UFTC A/B disk layout
LABEL=ROOT_A  /      ext4  errors=remount-ro  0  1
LABEL=ESP     /boot  vfat  umask=0077         0  2
tmpfs         /tmp   tmpfs defaults,noatime   0  0
FSTAB

# ---------------------------------------------------------------------------
# Install GRUB to the A/B disk using host grub-install
# ---------------------------------------------------------------------------
log "Installing GRUB"
ESP_PARTITION="$(partition_device "$AB_LOOP" 2)"
wait_for_block_device "$ESP_PARTITION" 30 || die "Partition node $ESP_PARTITION did not appear in time"
mount "$ESP_PARTITION" "$WORK_DIR/root_a/boot"

# BIOS GRUB (writes core image to the BIOSBOOT partition + modules to ESP/grub/)
grub-install \
    --target=i386-pc \
    --boot-directory="$WORK_DIR/root_a/boot" \
    --recheck \
    "$AB_LOOP"

# EFI GRUB (optional — skip if the chroot lacks the EFI target)
if [[ "$SKIP_EFI" == false ]]; then
    log "Installing EFI GRUB"
    grub-install \
        --target=x86_64-efi \
        --boot-directory="$WORK_DIR/root_a/boot" \
        --efi-directory="$WORK_DIR/root_a/boot" \
        --bootloader-id=uftc \
        --no-nvram \
    --recheck 2>/dev/null || log "EFI GRUB install skipped (target not available)"
fi

# ---------------------------------------------------------------------------
# Write GRUB configuration
# ---------------------------------------------------------------------------
log "Writing grub.cfg"
mkdir -p "$WORK_DIR/root_a/boot/grub"
cat > "$WORK_DIR/root_a/boot/grub/grub.cfg" <<'GRUBCFG'
insmod all_video
insmod gfxterm
set default="0"
set timeout=5
set timeout_style=menu

# Load persistent environment from the ESP
if [ -f ($root)/grub/grubenv ]; then
    load_env
fi

# Supply defaults when variables are absent (first boot, corrupt grubenv)
if [ -z "${current_slot}"    ]; then set current_slot=a;     fi
if [ -z "${previous_slot}"   ]; then set previous_slot=a;    fi
if [ -z "${boot_tries}"      ]; then set boot_tries=0;       fi
if [ -z "${pending_recovery}" ]; then set pending_recovery=false; fi

# If a staged update is waiting, boot into recovery to complete the swap
if [ "${pending_recovery}" = "true" ]; then
    set default="recovery"
    set timeout=3
else
    # Rollback: too many failed boot attempts → switch back to previous slot
    if [ "${boot_tries}" = "3" ]; then
        set current_slot="${previous_slot}"
        set boot_tries=0
        save_env current_slot boot_tries
    fi

    # Select the correct slot menu entry
    if [ "${current_slot}" = "b" ]; then
        set default="slot_b"
    else
        set default="slot_a"
    fi

    # Increment the boot attempt counter (skip if already committed = 99)
    if [ "${boot_tries}" != "99" ]; then
        if   [ "${boot_tries}" = "0" ]; then set boot_tries=1; save_env boot_tries;
        elif [ "${boot_tries}" = "1" ]; then set boot_tries=2; save_env boot_tries;
        elif [ "${boot_tries}" = "2" ]; then set boot_tries=3; save_env boot_tries;
        fi
    fi
fi

menuentry "UFTC (Slot A)" --id slot_a {
    search --no-floppy --label --set=root ESP
    linux  /vmlinuz-a root=LABEL=ROOT_A rw quiet loglevel=3
    initrd /initrd-a.img
}

menuentry "UFTC (Slot B)" --id slot_b {
    search --no-floppy --label --set=root ESP
    linux  /vmlinuz-b root=LABEL=ROOT_B rw quiet loglevel=3
    initrd /initrd-b.img
}

menuentry "UFTC Recovery" --id recovery {
    search --no-floppy --label --set=root RECOVERY
    linux  /boot/vmlinuz root=LABEL=RECOVERY rw quiet init=/usr/bin/recovery
    initrd /boot/initrd.img
}
GRUBCFG

if [[ "$SKIP_EFI" == false ]]; then
    log "Writing EFI GRUB chainload stubs"
    mkdir -p "$WORK_DIR/root_a/boot/EFI/uftc" "$WORK_DIR/root_a/boot/EFI/BOOT"

    cat > "$WORK_DIR/root_a/boot/EFI/uftc/grub.cfg" <<'EFI_GRUBCFG'
search --no-floppy --label --set=esp ESP
set prefix=($esp)/grub
configfile ($esp)/grub/grub.cfg
EFI_GRUBCFG

    cat > "$WORK_DIR/root_a/boot/EFI/BOOT/grub.cfg" <<'EFI_FALLBACK_CFG'
search --no-floppy --label --set=esp ESP
set prefix=($esp)/grub
configfile ($esp)/grub/grub.cfg
EFI_FALLBACK_CFG

    if [[ -f "$WORK_DIR/root_a/boot/EFI/uftc/grubx64.efi" ]]; then
        cp "$WORK_DIR/root_a/boot/EFI/uftc/grubx64.efi" "$WORK_DIR/root_a/boot/EFI/BOOT/BOOTX64.EFI"
    else
        log "WARNING: EFI binary not found at EFI/uftc/grubx64.efi; fallback BOOTX64.EFI was not created"
    fi
fi

# ---------------------------------------------------------------------------
# Initialise grubenv
# ---------------------------------------------------------------------------
log "Initialising grubenv"
grub-editenv "$WORK_DIR/root_a/boot/grub/grubenv" create
grub-editenv "$WORK_DIR/root_a/boot/grub/grubenv" set \
    current_slot=a \
    previous_slot=a \
    boot_tries=0 \
    pending_recovery=false \
    pending_slot=a

# Create /boot/slots/ directory for slot version tracking
mkdir -p "$WORK_DIR/root_a/boot/slots"

# ---------------------------------------------------------------------------
# Unmount boot mount
# ---------------------------------------------------------------------------
umount "$WORK_DIR/root_a/boot"

# ---------------------------------------------------------------------------
# Unmount all remaining mount points
# ---------------------------------------------------------------------------
umount "$WORK_DIR/recovery"
umount "$WORK_DIR/root_a"
umount "$WORK_DIR/esp"

if [[ "$PARTITION_TOOL" == "kpartx" ]]; then
    kpartx -d "$AB_LOOP" >/dev/null 2>&1 || true
fi

losetup -d "$AB_LOOP"; AB_LOOP=""

log "A/B disk image ready: $OUTPUT_DISK"
ls -lh "$OUTPUT_DISK"
