#!/usr/bin/env bash
set -euo pipefail

# BuildKit enables parallel layer execution, cache mounts, and better layer caching.
export DOCKER_BUILDKIT=1

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

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

IMAGE_NAME="uftc"
OUTPUT_VHD="uftc.vhd"
FORCE_OVERWRITE=0
SKIP_DOCKER_BUILD=0
NO_STAGING=0
STAGING_DIR=""
AUTO_STAGING_DIR=""

usage() {
	cat <<'EOF'
Usage: ./build.sh [options] [-- <d2vm args>]

Builds the UFTC Docker image, then converts it to a bootable VHD.

Options:
	-o, --output PATH          Output VHD path (default: uftc.vhd)
			--image-name NAME      Docker image name/tag base (default: uftc)
			--skip-docker-build    Reuse existing Docker image and only run d2vm convert
			--staging-dir PATH     Run d2vm conversion in PATH, then copy result to --output
			--no-staging           Disable automatic /mnt performance staging
	-f, --force                Overwrite an existing output VHD
	-h, --help                 Show this help message

Any arguments after -- are passed directly to d2vm convert.
EOF
}

D2VM_ARGS=()
while [[ $# -gt 0 ]]; do
	case "$1" in
		-o|--output)
			if [[ $# -lt 2 ]]; then
				echo "Missing value for $1" >&2
				exit 1
			fi
			OUTPUT_VHD="$2"
			shift 2
			;;
		--image-name)
			if [[ $# -lt 2 ]]; then
				echo "Missing value for $1" >&2
				exit 1
			fi
			IMAGE_NAME="$2"
			shift 2
			;;
		--skip-docker-build)
			SKIP_DOCKER_BUILD=1
			shift
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
		-f|--force)
			FORCE_OVERWRITE=1
			shift
			;;
		-h|--help)
			usage
			exit 0
			;;
		--)
			shift
			D2VM_ARGS+=("$@")
			break
			;;
		*)
			D2VM_ARGS+=("$1")
			shift
			;;
	esac
done

if [[ -f "$OUTPUT_VHD" ]] && [[ "$FORCE_OVERWRITE" != "1" ]]; then
	echo "Output already exists: $OUTPUT_VHD" >&2
	echo "Use --force to overwrite, or choose a different --output path." >&2
	exit 1
fi

if [[ "$SKIP_DOCKER_BUILD" != "1" ]]; then
	run_with_heartbeat "Building Docker image ($IMAGE_NAME)" sudo docker build . -t "$IMAGE_NAME"
else
	log_phase "Skipping Docker build; reusing image ${IMAGE_NAME}:latest"
fi

if [[ -f "$OUTPUT_VHD" ]]; then
	log_phase "Removing existing output VHD: $OUTPUT_VHD"
	rm -f "$OUTPUT_VHD"
fi

if [[ -z "$STAGING_DIR" ]] && [[ "$NO_STAGING" != "1" ]] && [[ "$SCRIPT_DIR" == /mnt/* ]]; then
	AUTO_STAGING_DIR="$(mktemp -d /var/tmp/uftc-d2vm.XXXXXX)"
	STAGING_DIR="$AUTO_STAGING_DIR"
	log_phase "Detected /mnt workspace. Using fast staging at $STAGING_DIR for d2vm conversion"
fi

if [[ -n "$STAGING_DIR" ]]; then
	mkdir -p "$STAGING_DIR"
	staged_output_name="$(basename "$OUTPUT_VHD")"
	if [[ -f "$STAGING_DIR/$staged_output_name" ]]; then
		rm -f "$STAGING_DIR/$staged_output_name"
	fi

	(
		cd "$STAGING_DIR"
		run_with_heartbeat "Converting Docker image to VHD via d2vm" \
			sudo "$SCRIPT_DIR/d2vm" convert "${IMAGE_NAME}:latest" -o "$staged_output_name" --bootloader grub --boot-size 4000 --size 14G --network-manager none "${D2VM_ARGS[@]}"
	)

	log_phase "Copying staged VHD to destination: $OUTPUT_VHD"
	mv -f "$STAGING_DIR/$staged_output_name" "$OUTPUT_VHD"

	if [[ -n "$AUTO_STAGING_DIR" ]]; then
		rmdir "$AUTO_STAGING_DIR" 2>/dev/null || true
	fi
else
	run_with_heartbeat "Converting Docker image to VHD via d2vm" \
		sudo ./d2vm convert "${IMAGE_NAME}:latest" -o "$OUTPUT_VHD" --bootloader grub --boot-size 4000 --size 14G --network-manager none "${D2VM_ARGS[@]}"
fi

log_phase "Build complete: $OUTPUT_VHD"
