#!/usr/bin/env bash
set -euo pipefail

# Run the full build as root to avoid piecemeal sudo permissions for loop/mount operations.
if [[ "$EUID" -ne 0 ]]; then
    if ! command -v sudo >/dev/null 2>&1; then
        echo "This script requires root privileges and sudo is not installed." >&2
        echo "Run as root, or install sudo and rerun." >&2
        exit 1
    fi
    exec sudo -E "$0" "$@"
fi

export DOCKER_BUILDKIT=1

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

BUILD_AB_DISK_SCRIPT="$SCRIPT_DIR/build-ab-disk.sh"

log_phase() {
    printf '\n[%s] %s\n' "$(date +'%H:%M:%S')" "$1"
}

run_with_heartbeat() {
    local label="$1"
    shift

    log_phase "$label"
    "$@" &
    local cmd_pid=$!
    local started_at
    started_at="$(date +%s)"

    while kill -0 "$cmd_pid" 2>/dev/null; do
        sleep 15
        if kill -0 "$cmd_pid" 2>/dev/null; then
            local now elapsed
            now="$(date +%s)"
            elapsed=$((now - started_at))
            printf '[%s] %s still running (%ss elapsed)\n' "$(date +'%H:%M:%S')" "$label" "$elapsed"
        fi
    done

    wait "$cmd_pid"
}

docker_build_with_retry() {
    local label="$1"
    shift
    local attempt
    local max_attempts=2

    for attempt in $(seq 1 "$max_attempts"); do
        if run_with_heartbeat "$label (attempt ${attempt}/${max_attempts})" "$@"; then
            return 0
        fi

        if (( attempt < max_attempts )); then
            log_phase "Docker build failed. Pruning BuildKit cache and retrying once"
            docker builder prune -af >/dev/null 2>&1 || true
        fi
    done

    return 1
}

IMAGE_NAME="RDOS"
RECOVERY_IMAGE_NAME="RDOS-recovery"
OUTPUT_PROD_RAW="RDOS-prod.raw"
OUTPUT_RECOVERY_RAW="recovery.raw"
OUTPUT_AB="RDOS-ab.img"
OUTPUT_AB_ZST=""
AB_ZSTD_LEVEL=9
FORCE_OVERWRITE=0
SKIP_DOCKER_BUILD=0
NO_CACHE=0
BUILD_AB=0

PARTITION_TOOL="partscan"

usage() {
    cat <<'EOF'
Usage: ./build.sh [options]

Builds RDOS Docker images and assembles raw disk artifacts (no d2vm/VHD).

Options:
  -o, --output PATH          Output production raw disk path (default: RDOS-prod.raw)
      --output-prod-raw PATH Output production raw disk path (default: RDOS-prod.raw)
      --output-recovery-raw PATH
                             Output recovery raw disk path (default: recovery.raw)
      --image-name NAME      Production Docker image name/tag base (default: RDOS)
      --skip-docker-build    Reuse existing Docker images and only assemble raw artifacts
      --no-cache             Rebuild Docker images without cache
      --ab                   Also assemble A/B disk artifact (RDOS-ab.img)
      --output-ab PATH       Output A/B disk path (default: RDOS-ab.img)
            --output-ab-zst PATH   Output compressed A/B disk path (default: <output-ab>.zst)
            --zstd-level N         Compression level for A/B .zst (1-19, default: 9)
  -f, --force                Overwrite existing outputs
  -h, --help                 Show this help message
EOF
}

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

extract_image_rootfs() {
    local image_name="$1"
    local target_dir="$2"
    local cid

    cid=$(docker create "$image_name")
    docker export "$cid" | tar -xpf - -C "$target_dir"
    docker rm "$cid" >/dev/null
}

build_source_raw_image() {
    local image_name="$1"
    local output_raw="$2"
    local disk_size="$3"
    local boot_mb="$4"

    local work_dir loop_dev p1 p2 kernel_path initrd_path
    work_dir="$(mktemp -d /var/tmp/RDOS-src.XXXXXX)"
    loop_dev=""

    cleanup_source_raw() {
        set +e
        umount -l "$work_dir/boot" 2>/dev/null || true
        umount -l "$work_dir/root" 2>/dev/null || true
        if [[ "$PARTITION_TOOL" == "kpartx" ]] && [[ -n "$loop_dev" ]]; then
            kpartx -d "$loop_dev" >/dev/null 2>&1 || true
        fi
        [[ -n "$loop_dev" ]] && losetup -d "$loop_dev" 2>/dev/null || true
        rm -rf "$work_dir"
    }
    trap cleanup_source_raw RETURN

    rm -f "$output_raw"
    truncate -s "$disk_size" "$output_raw"

    sgdisk --zap-all "$output_raw" >/dev/null
    sgdisk \
        -n 1:2048:+"${boot_mb}"M -t 1:ef00 -c 1:BOOT \
        -n 2:0:0               -t 2:8300 -c 2:ROOT \
        "$output_raw" >/dev/null

    loop_dev=$(losetup --find --show --partscan "$output_raw")
    prepare_loop_partitions "$loop_dev"

    p1="$(partition_device "$loop_dev" 1)"
    p2="$(partition_device "$loop_dev" 2)"

    wait_for_block_device "$p1" 30 || { echo "Partition node $p1 did not appear in time" >&2; return 1; }
    wait_for_block_device "$p2" 30 || { echo "Partition node $p2 did not appear in time" >&2; return 1; }

    mkfs.fat -F32 "$p1" >/dev/null
    mkfs.ext4 -q "$p2"

    mkdir -p "$work_dir/boot" "$work_dir/root"
    mount "$p2" "$work_dir/root"
    mount "$p1" "$work_dir/boot"

    extract_image_rootfs "$image_name" "$work_dir/root"

    kernel_path=""
    initrd_path=""
    if [[ -d "$work_dir/root/boot" ]]; then
        kernel_path=$(find "$work_dir/root/boot" -maxdepth 1 -type f -name 'vmlinuz*' 2>/dev/null | sort -V | tail -1)
        initrd_path=$(find "$work_dir/root/boot" -maxdepth 1 -type f \( -name 'initrd*' -o -name 'initramfs*' \) 2>/dev/null | sort -V | tail -1)
    fi

    [[ -n "$kernel_path" ]] || { echo "Kernel not found in /boot for image $image_name (install a kernel package in the image build)" >&2; return 1; }
    [[ -n "$initrd_path" ]] || { echo "Initrd/initramfs not found in /boot for image $image_name (install a kernel package that provides initramfs)" >&2; return 1; }

    cp "$kernel_path" "$work_dir/boot/vmlinuz"
    cp "$initrd_path" "$work_dir/boot/initrd.img"

    umount "$work_dir/boot"
    umount "$work_dir/root"

    if [[ "$PARTITION_TOOL" == "kpartx" ]]; then
        kpartx -d "$loop_dev" >/dev/null 2>&1 || true
    fi
    losetup -d "$loop_dev"
    loop_dev=""
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -o|--output|--output-prod-raw)
            [[ $# -lt 2 ]] && { echo "Missing value for $1" >&2; exit 1; }
            OUTPUT_PROD_RAW="$2"
            shift 2
            ;;
        --output-recovery-raw)
            [[ $# -lt 2 ]] && { echo "Missing value for $1" >&2; exit 1; }
            OUTPUT_RECOVERY_RAW="$2"
            shift 2
            ;;
        --image-name)
            [[ $# -lt 2 ]] && { echo "Missing value for $1" >&2; exit 1; }
            IMAGE_NAME="$2"
            shift 2
            ;;
        --skip-docker-build)
            SKIP_DOCKER_BUILD=1
            shift
            ;;
        --no-cache)
            NO_CACHE=1
            shift
            ;;
        --ab)
            BUILD_AB=1
            shift
            ;;
        --output-ab)
            [[ $# -lt 2 ]] && { echo "Missing value for $1" >&2; exit 1; }
            OUTPUT_AB="$2"
            shift 2
            ;;
        --output-ab-zst)
            [[ $# -lt 2 ]] && { echo "Missing value for $1" >&2; exit 1; }
            OUTPUT_AB_ZST="$2"
            shift 2
            ;;
        --zstd-level)
            [[ $# -lt 2 ]] && { echo "Missing value for $1" >&2; exit 1; }
            AB_ZSTD_LEVEL="$2"
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

if [[ -z "$OUTPUT_AB_ZST" ]]; then
    OUTPUT_AB_ZST="${OUTPUT_AB}.zst"
fi

if ! [[ "$AB_ZSTD_LEVEL" =~ ^[0-9]+$ ]] || (( AB_ZSTD_LEVEL < 1 || AB_ZSTD_LEVEL > 19 )); then
    echo "Invalid --zstd-level: $AB_ZSTD_LEVEL (expected 1-19)" >&2
    exit 1
fi

for cmd in sgdisk mkfs.fat mkfs.ext4 losetup partprobe qemu-img rsync grub-install grub-editenv tar zstd; do
    command -v "$cmd" >/dev/null 2>&1 || { echo "Missing required command: $cmd" >&2; exit 1; }
done

if [[ "$SKIP_DOCKER_BUILD" != "1" ]]; then
    docker_args=()
    [[ "$NO_CACHE" == "1" ]] && docker_args+=(--no-cache)
    docker_build_with_retry "Building production Docker image ($IMAGE_NAME)" docker build "${docker_args[@]}" . -t "$IMAGE_NAME"

    if [[ "$BUILD_AB" == "1" ]]; then
        docker_build_with_retry "Building recovery Docker image ($RECOVERY_IMAGE_NAME)" docker build "${docker_args[@]}" -f Dockerfile.recovery . -t "$RECOVERY_IMAGE_NAME"
    fi
else
    log_phase "Skipping Docker build; reusing image ${IMAGE_NAME}:latest"
fi

if [[ -f "$OUTPUT_PROD_RAW" ]] && [[ "$FORCE_OVERWRITE" != "1" ]]; then
    echo "Output already exists: $OUTPUT_PROD_RAW" >&2
    echo "Use --force to overwrite, or choose a different --output path." >&2
    exit 1
fi

if [[ -f "$OUTPUT_PROD_RAW" ]]; then
    rm -f "$OUTPUT_PROD_RAW"
fi
run_with_heartbeat "Assembling production raw source image" build_source_raw_image "${IMAGE_NAME}:latest" "$OUTPUT_PROD_RAW" 14G 4000
log_phase "Production raw image ready: $OUTPUT_PROD_RAW"

if [[ "$BUILD_AB" == "1" ]]; then
    if [[ -f "$OUTPUT_RECOVERY_RAW" ]] && [[ "$FORCE_OVERWRITE" != "1" ]]; then
        echo "Output already exists: $OUTPUT_RECOVERY_RAW" >&2
        echo "Use --force to overwrite, or choose a different --output-recovery-raw path." >&2
        exit 1
    fi

    if [[ -f "$OUTPUT_AB" ]] && [[ "$FORCE_OVERWRITE" != "1" ]]; then
        echo "Output already exists: $OUTPUT_AB" >&2
        echo "Use --force to overwrite, or choose a different --output-ab path." >&2
        exit 1
    fi

    if [[ -f "$OUTPUT_AB_ZST" ]] && [[ "$FORCE_OVERWRITE" != "1" ]]; then
        echo "Output already exists: $OUTPUT_AB_ZST" >&2
        echo "Use --force to overwrite, or choose a different --output-ab-zst path." >&2
        exit 1
    fi

    [[ -f "$OUTPUT_RECOVERY_RAW" ]] && rm -f "$OUTPUT_RECOVERY_RAW"
    [[ -f "$OUTPUT_AB" ]] && rm -f "$OUTPUT_AB"
    [[ -f "$OUTPUT_AB_ZST" ]] && rm -f "$OUTPUT_AB_ZST"
    [[ -f "${OUTPUT_AB_ZST}.size" ]] && rm -f "${OUTPUT_AB_ZST}.size"

    run_with_heartbeat "Assembling recovery raw source image" build_source_raw_image "${RECOVERY_IMAGE_NAME}:latest" "$OUTPUT_RECOVERY_RAW" 2G 200
    log_phase "Recovery raw image ready: $OUTPUT_RECOVERY_RAW"

    run_with_heartbeat "Assembling A/B+Recovery disk" \
        "$BUILD_AB_DISK_SCRIPT" \
            --prod-raw "$OUTPUT_PROD_RAW" \
            --recovery-raw "$OUTPUT_RECOVERY_RAW" \
            --output "$OUTPUT_AB"

    log_phase "A/B disk ready: $OUTPUT_AB"

    run_with_heartbeat "Compressing A/B disk artifact (zstd level ${AB_ZSTD_LEVEL})" \
        zstd -T0 "-${AB_ZSTD_LEVEL}" -f "$OUTPUT_AB" -o "$OUTPUT_AB_ZST"
    stat -c%s "$OUTPUT_AB" > "${OUTPUT_AB_ZST}.size"
    log_phase "Compressed A/B disk ready: $OUTPUT_AB_ZST"
fi
