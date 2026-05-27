#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export RDOS_UI_LISTEN="${RDOS_UI_LISTEN:-127.0.0.1:8080}"
export RDOS_TCCONFIG="${RDOS_TCCONFIG:-$ROOT_DIR/.dev/tcconfig}"
export RDOS_SESSION_LOG="${RDOS_SESSION_LOG:-$ROOT_DIR/.dev/session.log}"

mkdir -p "$(dirname "$RDOS_TCCONFIG")"
touch "$RDOS_TCCONFIG"

exec go run ./cmd/thinclient-go
