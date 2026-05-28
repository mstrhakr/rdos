# Go and Web UI MVP

This document describes the first implementation slice for the RDOS Go backend and fullscreen web UI.

## Included in this slice

- Go service: `cmd/thinclient-go`
- Embedded static UI from `cmd/thinclient-go/web`
- API endpoints:
  - `GET /api/v1/health`
  - `GET /api/v1/config`
  - `POST /api/v1/config`
  - `GET /api/v1/ota`
  - `POST /api/v1/ota/rollback`
  - `GET /api/v1/session`
  - `POST /api/v1/session/connect`
  - `POST /api/v1/session/disconnect`
- tcconfig compatibility parser/writer in `internal/tcconfig`
- xfreerdp3 launch manager in `internal/session`
- Boot mode detection from kernel cmdline (`rdos.ui=web|legacy`) via `internal/bootmode`

## Boot mode integration

- RDOS now supports two UI boot modes using the same image/rootfs:
  - `rdos.ui=web` launches fullscreen web UI (`thinclient-go` + Chromium kiosk)
  - `rdos.ui=legacy` launches existing shell/YAD flow (`/usr/bin/thinclient`)
- The mode is selected by GRUB menu entries in both VHD and A/B image builders.
- Runtime launcher: `/usr/bin/tc-ui-launch`.
- Web mode is web-only; it does not hand off to the legacy shell flow.

## Out of scope in this slice

- No systemd unit wiring yet
- No kiosk browser launch integration yet
- No manual update execution or timer control yet
- No replacement of existing shell UI path yet
- No separate app-layer updater; RDOS stays on a single deployable version with A/B image rollback for platform updates

## Next parity target

- OTA management is the next feature-parity gap to close: network, WiFi, and WireGuard surfaces are already present in `thinclient-go`, and the first OTA slice now covers live slot status plus rollback entry, but update execution and timer control are still missing.

## Local run

```bash
./scripts/run-thinclient-go.sh
```

Then open `http://127.0.0.1:8080`.

## Notes

- API access is restricted to loopback clients in this MVP.
- This service is additive and does not modify current RDOS startup behavior.
- OTA status and rollback are now exposed in the web settings UI, but update execution and timer management still remain to be wired.
- Platform update flow remains A/B image based; the web UI is not a second independently versioned application.
