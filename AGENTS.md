# RDOS Agent Instructions

RDOS (User Friendly ThinClient) is a lightweight Linux thin-client distribution for RDP deployments. This file helps AI agents understand the project structure, build system, and development conventions.

## Quick Reference

**Build Environment**: Linux or WSL2 (not Windows native—line endings and permissions are critical)

**Build Command**: `./build.sh` → produces `RDOS.vhd` (14GB bootable disk image)

**ISO Build**: `./build-installer-iso.sh` → produces unattended installer ISO

**First Boot**: VHD requires one VM boot to finalize (runs `firstboot`, refreshes initramfs, installs XanMod kernel only if missing, resets hostname)

## Project Overview

- **Purpose**: Locked-down, user-friendly Linux thin client for RDP (FreeRDP 3 + FVWM UI)
- **Base OS**: Debian Trixie
- **Kernel**: XanMod LTS x64v1 (installed at first boot for device compatibility)
- **Build System**: Docker → [d2vm](d2vm) (privileged container) → bootable VHD
- **Deployment**: VHD directly to VM or via unattended Clonezilla installer ISO

## Build Architecture

### Two-Stage Build

1. **Docker build** (`Dockerfile`) assembles Debian-based filesystem with packages, firmware, and services
2. **d2vm convert** (privileged container) transforms Docker image into 14GB VHD with GRUB bootloader

### Important Constraints

- **d2vm is privileged**: Requires `--privileged` flag and Docker socket access
- **Deferred initialization**: Kernel and initramfs installed at first VM boot, not during build (docker can't do this)
- **BuildKit enabled**: Uses cache mounts for faster rebuilds (apt cache persists)

### Why Linux/WSL2?

- Line endings: Repository enforces LF via `.gitattributes` (CRLF breaks shell scripts)
- File permissions: Executable bits and special permissions must be preserved
- d2vm behavior: Works reliably in Linux containers; problematic on Windows native

## Critical Files & Customization Points

### Core Boot & UI

- **[tcfiles/thinclient](tcfiles/thinclient)** – Main launcher script
  - Handles RDP connection, first-boot completion check, dynamic hostname from MAC
  - Loads config from `/boot/tcconfig` using safe key=value parsing (not bash `source`)
  
- **[tcfiles/firstboot](tcfiles/firstboot)** – Second-phase initialization (runs at first VM boot)
  - Installs XanMod kernel only when needed, regenerates initramfs, resets hostname, self-deletes
  - Do not modify lightly—failures here require rebuild and re-boot

- **[tcfiles/.fvwm/config](tcfiles/.fvwm/config)** – Window manager UI (minimal menu system, logout, shutdown buttons)

### System Services

- **[tcfiles/tc-copyconfig.service](tcfiles/tc-copyconfig.service)** – Injects config files from `/boot` at boot
- **[tcfiles/tc-copywpa.service](tcfiles/tc-copywpa.service)** – Injects WiFi credentials
- **[tcfiles/tc-wifipower.service](tcfiles/tc-wifipower.service)** – WiFi power management
- **`systemd-networkd` + `systemd-resolved`** – Primary wired DHCP, WiFi DHCP, routing, and DNS stack

### Configuration

- **[tcfiles/debian.sources](tcfiles/debian.sources)** – APT sources (includes XanMod repo)
- **[Dockerfile](Dockerfile)** – Package selection, firmware bundles (iwlwifi, realtek, atheros, broadcom)
- **[tcfiles/dhcp.network](tcfiles/dhcp.network)** – systemd-networkd DHCP policy for wired and wireless adapters
- **[tcfiles/systemd-resolved.conf](tcfiles/systemd-resolved.conf)** – tmpfiles rule that links `/etc/resolv.conf` to the resolved stub
- **[tcfiles/firstboot](tcfiles/firstboot)** – Enables services, sets autologin, configures systemd

### Optional Customization

- **[tcfiles/xorg.conf](tcfiles/xorg.conf)** – Display server config (pre-optimized for RDP)
- **[tcfiles/dhcp.network](tcfiles/dhcp.network)** – Systemd-networkd DHCP config

## Development Workflow

### Safe Test Loop

```bash
1. Edit files in tcfiles/ or Dockerfile
2. ./build.sh
3. Boot RDOS.vhd in a VM
4. Let first boot complete (watch for poweroff)
5. Boot again, validate RDP login/connectivity
```

### Build Troubleshooting

- **Permission denied errors**: Likely line-ending issue. Ensure repo is cloned on Linux filesystem or WSL2
- **Initramfs errors**: If `firstboot` fails, the VHD won't boot next time. Check logs in first boot console output
- **d2vm failures**: Check Docker daemon, ensure `--privileged` and socket mounting are correct in [d2vm](d2vm)

## Configuration & Customization Patterns

### Runtime Config (First Boot)

Config files injected to `/boot/tcconfig` are loaded by `thinclient` using safe key=value parsing:

```
adminpass=mysecretpassword
keylayout=en_US
rdpserver=myserver.example.com
rdpusername=username
autoconnect=true
exit_type=Poweroff
```

See [tcfiles/thinclient](tcfiles/thinclient) for supported keys and `load_config()` function.

### Hostname Configuration

Machines auto-rename to `TC-{MAC_ADDRESS}` on first boot (dynamic hostname). To override, set `static_hostname` in `/boot/tcconfig` during first boot.

### Image Baking vs. Runtime Injection

- **Baked into image**: Place files in `tcfiles/` (persists across deployments)
- **Runtime injection**: Place in `/boot/tcconfig` (one-time, deployment-specific)

## Common Conventions

### Code Style

- Shell scripts: Bash 4+ (POSIX-ish), LF line endings
- Safe config loading: Use key=value parsing, NOT `bash source` (security risk)
- Systemd services: Standard [Unit]/[Service] format

### Testing

- No automated tests; validation via VM boot cycles (see [docs/building.md](docs/building.md))
- Always boot twice: first boot runs `firstboot`, second boot validates actual thin-client behavior

### Versioning

- [Version](Version) file tracks image version (simple text file)
- ISO builds reference this for naming

## Troubleshooting Checklist

1. **"Permission denied" or script failures**: Check line endings (should be LF)
2. **First boot hangs or fails**: Check `firstboot` script, likely initramfs generation issue
3. **VHD won't boot after modifications**: May need to rebuild and first-boot again
4. **d2vm errors**: Ensure Docker daemon running, check [d2vm](d2vm) wrapper script, verify `--privileged`
5. **WiFi or device issues**: XanMod kernel (installed at first boot) provides broad compatibility; check firmware bundles in Dockerfile

## Key References

- [docs/building.md](docs/building.md) – Detailed build process and safe test loop
- [Readme.md](Readme.md) – Project overview and usage
- [docs/troubleshooting.md](docs/troubleshooting.md) – User-facing troubleshooting guide

---

**Updated**: 2026-05-17
