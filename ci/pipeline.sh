#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

TARGET_STEP="all"
NO_DEPS=0
RUN_MODE="local"
SHELLCHECK_SEVERITY=""

OUTPUT_AB="rdos-ab.img"
OUTPUT_AB_ZST="rdos-ab.img.zst"
OUTPUT_ISO="rdos-installer.iso"
OUTPUT_PROD_RAW="rdos-prod.raw"
OUTPUT_VHDX="rdos.vhdx"
BUILD_IMG_MODE="auto"

NO_CACHE=0
NO_STAGING=0
STAGING_DIR=""

usage() {
  cat <<'EOF'
Usage: ./ci/pipeline.sh [target] [options]

Targets:
  pretest    Run preflight validation only
  build-img  Build image artifact only (build.sh)
  build-vhdx Convert production raw image to VHDX (dev helper)
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
  --output-prod-raw PATH
                       Production raw artifact path (default: rdos-prod.raw)
  --output-vhdx PATH   VHDX artifact path for build-vhdx target (default: rdos.vhdx)
  --output-ab PATH     A/B disk artifact path (default: rdos-ab.img)
  --output-ab-zst PATH Compressed A/B disk artifact path (default: rdos-ab.img.zst)
  --output-iso PATH    ISO artifact path (default: rdos-installer.iso)
  --build-img-mode MODE
                       Build mode for build.sh image assembly:
                       auto (default), prod, both
  --no-cache           Pass no-cache to build scripts
  --no-staging         Pass no-staging to build scripts
  --staging-dir PATH   Pass staging-dir to build scripts
  -h, --help           Show this help

Examples:
  ./ci/pipeline.sh all
  ./ci/pipeline.sh all --build-img-mode both
  ./ci/pipeline.sh build-vhdx
  ./ci/pipeline.sh build-vhdx --build-img-mode prod
  ./ci/pipeline.sh build-vhdx --output-prod-raw rdos-prod.raw --output-vhdx rdos.vhdx
  ./ci/pipeline.sh iso-test
  ./ci/pipeline.sh iso-test --no-deps
  ./ci/pipeline.sh build-iso --output-ab rdos-ab.img --output-ab-zst rdos-ab.img.zst --output-iso rdos-installer.iso
EOF
}

if [[ $# -gt 0 ]]; then
  case "$1" in
    pretest|build-img|build-vhdx|img-test|ab-test|build-iso|iso-test|all)
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
      NO_DEPS=1
      shift
      ;;
    --output-prod-raw)
      [[ $# -lt 2 ]] && { echo "Missing value for --output-prod-raw" >&2; exit 1; }
      OUTPUT_PROD_RAW="$2"
      shift 2
      ;;
    --output-vhdx)
      [[ $# -lt 2 ]] && { echo "Missing value for --output-vhdx" >&2; exit 1; }
      OUTPUT_VHDX="$2"
      shift 2
      ;;
    --output-ab)
      [[ $# -lt 2 ]] && { echo "Missing value for --output-ab" >&2; exit 1; }
      OUTPUT_AB="$2"
      shift 2
      ;;
    --output-ab-zst)
      [[ $# -lt 2 ]] && { echo "Missing value for --output-ab-zst" >&2; exit 1; }
      OUTPUT_AB_ZST="$2"
      shift 2
      ;;
    --output-iso)
      [[ $# -lt 2 ]] && { echo "Missing value for --output-iso" >&2; exit 1; }
      OUTPUT_ISO="$2"
      shift 2
      ;;
    --build-img-mode)
      [[ $# -lt 2 ]] && { echo "Missing value for --build-img-mode" >&2; exit 1; }
      BUILD_IMG_MODE="$2"
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

if [[ "$BUILD_IMG_MODE" != "auto" && "$BUILD_IMG_MODE" != "prod" && "$BUILD_IMG_MODE" != "both" ]]; then
  echo "Invalid --build-img-mode: $BUILD_IMG_MODE (expected auto, prod, or both)" >&2
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

resolve_build_img_mode() {
  local requested_mode="$1"
  local target_step="$2"

  if [[ "$requested_mode" == "prod" || "$requested_mode" == "both" ]]; then
    printf '%s' "$requested_mode"
    return 0
  fi

  case "$target_step" in
    build-vhdx)
      printf '%s' "prod"
      ;;
    *)
      printf '%s' "both"
      ;;
  esac
}

run_build_img() {
  local mode="$1"
  local args=(--output "$OUTPUT_PROD_RAW" --force)
  [[ "$NO_CACHE" == "1" ]] && args+=(--no-cache)

  if [[ "$mode" == "both" ]]; then
    args+=(--output-ab "$OUTPUT_AB" --output-ab-zst "$OUTPUT_AB_ZST" --ab)
  fi

  if [[ "$mode" == "both" ]]; then
    log "Step build-img: building production, recovery, and A/B disk artifacts"
  else
    log "Step build-img: building production raw source artifact only"
  fi

  bash ./build.sh "${args[@]}"
}

run_build_vhdx() {
  log "Step build-vhdx: converting production raw image to VHDX"
  bash ./ci/convert-raw-to-vhdx.sh --input "$OUTPUT_PROD_RAW" --output "$OUTPUT_VHDX" --force
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
  local args=(--skip-build --input-disk-zst "$OUTPUT_AB_ZST" --output-iso "$OUTPUT_ISO")
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

run_step() {
  local step="$1"
  local mode=""
  case "$step" in
    pretest) run_pretest ;;
    build-img)
      mode="$(resolve_build_img_mode "$BUILD_IMG_MODE" "$TARGET_STEP")"
      run_build_img "$mode"
      ;;
    build-vhdx) run_build_vhdx ;;
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
  run_step pretest
  run_step build-img
  run_step ab-test
  run_step build-iso
  run_step iso-test
elif [[ "$NO_DEPS" == "0" ]]; then
  case "$TARGET_STEP" in
    pretest)
      run_step pretest
      ;;
    build-img)
      run_step pretest
      run_step build-img
      ;;
    build-vhdx)
      run_step pretest
      run_step build-img
      run_step build-vhdx
      ;;
    ab-test)
      run_step pretest
      run_step build-img
      run_step ab-test
      ;;
    build-iso)
      run_step pretest
      run_step build-img
      run_step ab-test
      run_step build-iso
      ;;
    iso-test)
      run_step pretest
      run_step build-img
      run_step ab-test
      run_step build-iso
      run_step iso-test
      ;;
    *)
      echo "Invalid target: $TARGET_STEP" >&2
      usage >&2
      exit 1
      ;;
  esac
else
  run_step "$TARGET_STEP"
fi

log "Pipeline target complete: $TARGET_STEP"
