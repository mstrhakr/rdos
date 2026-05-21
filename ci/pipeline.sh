#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

TARGET_STEP="all"
WITH_DEPS=1
RUN_MODE="local"
SHELLCHECK_SEVERITY=""

OUTPUT_AB="uftc-ab.img"
OUTPUT_ISO="uftc-installer.iso"

NO_CACHE=0
NO_STAGING=0
STAGING_DIR=""

usage() {
  cat <<'EOF'
Usage: ./ci/pipeline.sh [target] [options]

Targets:
  pretest    Run preflight validation only
  build-img  Build image artifact only (build.sh)
  img-test   Validate image artifact only (alias for ab-test)
  ab-test    Validate A/B disk artifact only
  build-iso  Build installer ISO only (build-installer-iso.sh)
  iso-test   Validate installer ISO only
  all        Run full sequence (default)

Options:
  --mode MODE          local (default) or ci (passed to preflight)
  --shellcheck-severity LEVEL
                       Optional override for preflight shellcheck threshold
                       (error|warning|info|style)
  --no-deps            Run only selected target without earlier steps
  --with-deps          Run selected target with prerequisite steps (default)
  --output-ab PATH     A/B disk artifact path (default: uftc-ab.img)
  --output-iso PATH    ISO artifact path (default: uftc-installer.iso)
  --no-cache           Pass no-cache to build scripts
  --no-staging         Pass no-staging to build scripts
  --staging-dir PATH   Pass staging-dir to build scripts
  -h, --help           Show this help

Examples:
  ./ci/pipeline.sh all
  ./ci/pipeline.sh iso-test
  ./ci/pipeline.sh iso-test --no-deps
  ./ci/pipeline.sh build-iso --with-deps --output-ab uftc-ab.img --output-iso uftc-installer.iso
EOF
}

if [[ $# -gt 0 ]]; then
  case "$1" in
    pretest|build-img|img-test|ab-test|build-iso|iso-test|all)
      TARGET_STEP="$1"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
  esac
fi

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      [[ $# -lt 2 ]] && { echo "Missing value for --mode" >&2; exit 1; }
      RUN_MODE="$2"
      shift 2
      ;;
    --shellcheck-severity)
      [[ $# -lt 2 ]] && { echo "Missing value for --shellcheck-severity" >&2; exit 1; }
      SHELLCHECK_SEVERITY="$2"
      shift 2
      ;;
    --no-deps)
      WITH_DEPS=0
      shift
      ;;
    --with-deps)
      WITH_DEPS=1
      shift
      ;;
    --output-ab)
      [[ $# -lt 2 ]] && { echo "Missing value for --output-ab" >&2; exit 1; }
      OUTPUT_AB="$2"
      shift 2
      ;;
    --output-iso)
      [[ $# -lt 2 ]] && { echo "Missing value for --output-iso" >&2; exit 1; }
      OUTPUT_ISO="$2"
      shift 2
      ;;
    --no-cache)
      NO_CACHE=1
      shift
      ;;
    --no-staging)
      NO_STAGING=1
      shift
      ;;
    --staging-dir)
      [[ $# -lt 2 ]] && { echo "Missing value for --staging-dir" >&2; exit 1; }
      STAGING_DIR="$2"
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

if [[ "$RUN_MODE" != "local" && "$RUN_MODE" != "ci" ]]; then
  echo "Invalid --mode: $RUN_MODE (expected local or ci)" >&2
  exit 1
fi

if [[ "$TARGET_STEP" == "img-test" ]]; then
  TARGET_STEP="ab-test"
fi

log() {
  printf '[%s] %s\n' "$(date +'%H:%M:%S')" "$1"
}

run_pretest() {
  local args=(--mode "$RUN_MODE" --path preflight)
  if [[ -n "$SHELLCHECK_SEVERITY" ]]; then
    args+=(--shellcheck-severity "$SHELLCHECK_SEVERITY")
  fi

  log "Step pretest: running preflight validation"
  bash ./ci/preflight-validate.sh "${args[@]}"
}

run_build_img() {
  local args=(--output-ab "$OUTPUT_AB" --ab --force)
  [[ "$NO_CACHE" == "1" ]] && args+=(--no-cache)

  log "Step build-img: building raw source and A/B disk artifacts"
  bash ./build.sh "${args[@]}"
}

run_img_test() {
  log "Step img-test: validating A/B disk artifact"
  bash ./ci/ab-disk-validate.sh --output "$OUTPUT_AB"
}

run_ab_test() {
  log "Step ab-test: validating A/B disk artifact"
  bash ./ci/ab-disk-validate.sh --output "$OUTPUT_AB"
}

run_build_iso() {
  local args=(--skip-build --input-disk "$OUTPUT_AB" --output-iso "$OUTPUT_ISO")
  [[ "$NO_CACHE" == "1" ]] && args+=(--no-cache)
  [[ "$NO_STAGING" == "1" ]] && args+=(--no-staging)
  [[ -n "$STAGING_DIR" ]] && args+=(--staging-dir "$STAGING_DIR")

  log "Step build-iso: building installer ISO from A/B disk artifact"
  bash ./build-installer-iso.sh "${args[@]}"
}

run_iso_test() {
  log "Step iso-test: validating installer ISO artifact"
  bash ./ci/iso-build-validate.sh --mode validate-only --payload-layout ab --output-iso "$OUTPUT_ISO"
}

steps=(pretest build-img ab-test build-iso iso-test)
target_index=-1

if [[ "$TARGET_STEP" == "all" ]]; then
  target_index=$((${#steps[@]} - 1))
else
  for i in "${!steps[@]}"; do
    if [[ "${steps[$i]}" == "$TARGET_STEP" ]]; then
      target_index="$i"
      break
    fi
  done
fi

if (( target_index < 0 )); then
  echo "Invalid target: $TARGET_STEP" >&2
  usage >&2
  exit 1
fi

run_step() {
  local step="$1"
  case "$step" in
    pretest) run_pretest ;;
    build-img) run_build_img ;;
    img-test) run_img_test ;;
    ab-test) run_ab_test ;;
    build-iso) run_build_iso ;;
    iso-test) run_iso_test ;;
    *)
      echo "Unknown internal step: $step" >&2
      exit 1
      ;;
  esac
}

if [[ "$TARGET_STEP" == "all" ]]; then
  for step in "${steps[@]}"; do
    run_step "$step"
  done
elif [[ "$WITH_DEPS" == "1" ]]; then
  for i in "${!steps[@]}"; do
    if (( i <= target_index )); then
      run_step "${steps[$i]}"
    fi
  done
else
  run_step "$TARGET_STEP"
fi

log "Pipeline target complete: $TARGET_STEP"
