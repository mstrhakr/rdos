#!/usr/bin/env bash
set -euo pipefail

# BuildKit enables parallel layer execution, cache mounts, and better layer caching.
export DOCKER_BUILDKIT=1

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

IMAGE_NAME="uftc"
OUTPUT_VHD="uftc.vhd"
FORCE_OVERWRITE=0
SKIP_DOCKER_BUILD=0

usage() {
	cat <<'EOF'
Usage: ./build.sh [options] [-- <d2vm args>]

Builds the UFTC Docker image, then converts it to a bootable VHD.

Options:
	-o, --output PATH          Output VHD path (default: uftc.vhd)
			--image-name NAME      Docker image name/tag base (default: uftc)
			--skip-docker-build    Reuse existing Docker image and only run d2vm convert
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
	sudo docker build . -t "$IMAGE_NAME"
fi

if [[ -f "$OUTPUT_VHD" ]]; then
	rm -f "$OUTPUT_VHD"
fi

sudo ./d2vm convert "${IMAGE_NAME}:latest" -o "$OUTPUT_VHD" --bootloader grub --boot-size 4000 --size 14G --network-manager none "${D2VM_ARGS[@]}"
