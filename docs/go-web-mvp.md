# Go and Web UI MVP

This document describes the first implementation slice for the RDOS Go backend and fullscreen web UI.

## Included in this slice

- Go service: `cmd/thinclient-go`
- Embedded static UI from `cmd/thinclient-go/web`
- API endpoints:
  - `GET /api/v1/health`
  - `GET /api/v1/config`
  - `POST /api/v1/config`
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
- Fallback behavior: if web mode components fail to start, launcher falls back to legacy UI automatically.

## Out of scope in this slice

- No systemd unit wiring yet
- No kiosk browser launch integration yet
- No network and OTA management APIs yet
- No replacement of existing shell UI path yet
- No separate app-layer updater; RDOS stays on a single deployable version with A/B image rollback for platform updates

## Local run

```bash
./scripts/run-thinclient-go.sh
```

Then open `http://127.0.0.1:8080`.

## Notes

- API access is restricted to loopback clients in this MVP.
- This service is additive and does not modify current RDOS startup behavior.
- Platform update flow remains A/B image based; the web UI is not a second independently versioned application.
