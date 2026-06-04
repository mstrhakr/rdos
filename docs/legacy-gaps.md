# Legacy UI Feature Gaps

The bash/yad/FVWM legacy UI stack was removed in favour of the web UI
(`thinclient-go` + Chromium kiosk). This file records features that existed
in the legacy UI but are not yet implemented in the web UI, so they can be
re-added iteratively.

Removed files: `tcfiles/thinclient`, `tcfiles/tc-settings`,
`tcfiles/tc-wifi-wizard`, `tcfiles/tc-set-background`, `tcfiles/.fvwm/`,
`tcfiles/xinitrc`, `tcfiles/autologin`, `tcfiles/autologin-console`.

---

## Connection features

### Multi-server selection
`thinclient` accepted a pipe-separated `server` list (e.g.
`server=host1|host2|host3`) and showed a yad list dialog for the user to
pick one. The web UI only accepts a single server value.

### Remote config URL (`config_url`)
`thinclient` supported a `config_url` key that downloaded a fresh `tcconfig`
on every boot, with fallback to the last cached copy. This allowed centralised
config management without OTA. The web UI does not fetch or merge remote
configs.

### Citrix ICA integration
When `server=citrix`, `thinclient` launched
`/opt/Citrix/ICAClient/selfservice` and handled the `param` field as a store
URL. The web UI has no Citrix session type.

### Moonlight / game-streaming
When `server=moonlight`, `thinclient` launched `/usr/bin/moonlight` with
`param` as arguments. The web UI has no Moonlight session type.

### RDP file / URL launch
`thinclient` detected `://` in the `server` field, downloaded the URL as an
`.rdp` file with `curl`+`enconv`, and passed it directly to `xfreerdp3`.
The web UI only supports hostname/IP-style server values.

### Autologon with bypass timer
The `autologon=true` config key made `thinclient` skip the login form and
connect immediately, with a 1-second yad cancel window. If the user clicked
cancel, manual login was offered. The web UI has no autologon mode.

---

## Session / post-login features

### Friendly RDP error messages
`thinclient` mapped FreeRDP `ERRCONNECT_*` and `ERRINFO_*` log codes to
human-readable yad error dialogs (wrong password, server not found, account
locked, licence limit reached, etc.). The web UI shows the raw session log
and a generic error state without per-error messaging.

### Unknown host-key trust prompt
When FreeRDP reported a certificate verification error, `thinclient` offered
a yad dialog to trust and save the key, then reconnected automatically with
`/cert:tofu`. The web UI has no interactive cert-trust flow.

### Post-RDP reconnect vs shutdown logic
`thinclient` distinguished between a user-initiated sign-out and an
unexpected disconnect, restarting the login loop on disconnect and honouring
`exit_type` (Shutdown / Sleep / Restart / Exit) on sign-out with `autologon`.
The web UI restarts the session unconditionally.

---

## System settings (Device tab)

These config keys exist and are saved by the web UI settings panel but are
not applied at session startup because the code that applied them was removed
with `thinclient`.

### Volume and microphone initialisation
`pamixer --set-volume "$volume"` and `pamixer --default-source --set-volume
"$microphone"` were run on every login. The web UI settings panel saves these
keys but nothing applies them at startup.

### Brightness initialisation
`light -S "$brightness"` was run on every login. Requires the `light` package.
The web UI saves the key but nothing applies it.

### Screen standby timeout
`xset dpms 0 "$screen_timeout" 0` was run on every login. The web UI saves
the key but nothing applies it.

### Keyboard layout
`setxkbmap "$keylayout"` was run on every login. The web UI saves the key
but nothing applies it.

### Status overlay (yad-based)
`thinclient` ran a background `yad` progress widget showing time, battery,
WiFi, IP, and WireGuard status. This is partially superseded by
`tc-overlay-daemon` (Go-based X11 bar), but the yad overlay supported battery
low-threshold notifications which the Go daemon does not yet implement.

---

## Wallpaper

### Deployment wallpaper from `/boot`
`tc-set-background` read `wallpaper_mode` from tcconfig, searched
`/boot/wallpaper.*` and `~/wallpaper.*` first, then fell back to
`/usr/share/rdos/wallpaper-default.png`, and set the desktop background with
`feh`. With `feh`, FVWM, and `tc-set-background` removed, no wallpaper is
applied. The LightDM desktop is shown as-is.

`wallpaper_mode` values: `fill`, `max`/`fit`, `scale`/`stretch`, `center`,
`tile`.

---

## Admin / settings access

### Admin password gate on settings
`tc-settings` prompted for the `adminpass` config value before showing the
settings panel. The web UI exposes all settings without requiring a password.
The `adminpass` key still exists in tcconfig but is not enforced by the web UI.

### Admin password gate on USB config import
`tc-import-usb --interactive` (called from within an active session) prompted
for `adminpass` before applying a USB-sourced `tcconfig`. The web UI
`/api/v1/ota/usb` import path has no admin password check.

### First-boot wizard
`tc-settings --wizard` ran a guided multi-step yad wizard (connection →
WiFi → network → support → summary) when no `tcconfig` was present. The web
UI settings page is always available but there is no guided wizard mode for
unconfigured devices.

---

## Tools

### Network info dialog
The Tools section in `tc-settings` showed hostname, local IPs, public IP
(`curl ifconfig.me`), DNS servers, and default gateway in a yad dialog.
The web UI has no equivalent summary view.

### Ping/trace tool
The Tools section launched `mtr <target>` in an `xterm` with the RDP server
pre-filled as the default target. The web UI has a terminal (ttyd) where
`mtr` can be run manually, but there is no guided ping/trace tool.

---

## Packages removed from image

| Package | Was used for |
|---------|-------------|
| `fvwm` | Window manager for legacy session |
| `feh` | Desktop wallpaper setter |
| `light` | Hardware backlight control |

`yad` is **retained** — it is still used by `tc-import-usb` (admin password
prompt and USB import confirmation dialog).
