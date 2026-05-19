# Building UFTC

This repository builds:

- `uftc.vhd` (bootable virtual disk image)
- `uftc-installer.iso` (minimal unattended installer ISO)

The normal flow is still to build `uftc.vhd` first, then optionally package it into the installer ISO.

## What the build actually does

The current build flow is small but it depends on Linux container behavior:

1. `docker build . -t uftc`
2. `./d2vm convert uftc:latest -o uftc.vhd --bootloader grub --boot-size 4000 --size 14G --network-manager none`

In practice that means:

- [Dockerfile](Dockerfile) assembles a Debian-based thin-client filesystem.
- [build.sh](build.sh) builds the container and converts it into `uftc.vhd`.
- [d2vm](d2vm) is a wrapper around the `linkacloud/d2vm` container, which needs privileged access and the Docker socket.
- Runtime networking is provided by `systemd-networkd` and `systemd-resolved`, with `wpa_supplicant@<iface>.service` used for WiFi association.

## Recommended environment

If you are on Windows, use WSL2.

Recommended setup:

1. Install Docker Desktop.
2. Enable WSL2 integration for your Linux distro.
3. Clone this repo inside the Linux filesystem, not under a Windows-mounted path if you can avoid it.
4. Build from the Linux shell.

Why: the repo relies on Linux line endings, executable bits, and privileged container behavior. Those are easier to preserve inside WSL than on the Windows side.

On Windows, line endings are a common source of failures for shell scripts. This repo includes `.gitattributes` to enforce LF for Linux-consumed files.

## Build steps

From your Linux or WSL shell in the repo root:

```bash
chmod +x build.sh d2vm
./build.sh
```

Optional: avoid repeated sudo password prompts during local builds:

```bash
chmod +x setup-build-sudoers.sh
./setup-build-sudoers.sh
```

This creates `/etc/sudoers.d/uftc-build-nopasswd` for the current Linux user and allows passwordless sudo only for the local `docker` binary and this repo's `d2vm` wrapper.

To remove the rule later:

```bash
./setup-build-sudoers.sh --remove
```

If the build succeeds, you should get:

- `uftc.vhd`

## Preflight validation

Run preflight validation before image builds to catch script, line-ending, permission, and tooling issues early.

```bash
chmod +x ci/preflight-validate.sh
./ci/preflight-validate.sh --mode local --path preflight
```

ShellCheck defaults are tuned for signal over noise:

- `local` mode fails on ShellCheck `error`
- `ci` mode fails on ShellCheck `warning` and above

Override severity if needed:

```bash
./ci/preflight-validate.sh --mode ci --path preflight --shellcheck-severity error
```

Path scopes:

- `preflight`: structural checks only, tool checks are advisory in local mode
- `vhd`: validates structural checks and requires VHD-path tooling
- `iso`: validates structural checks and requires ISO-path tooling
- `all`: validates structural checks and requires both tooling sets

CI should call the same script in `ci` mode and choose a path scope per job matrix leg.

## VHD build validation

Use the canonical VHD validation wrapper to run `build.sh` and assert artifact shape.

```bash
chmod +x ci/vhd-build-validate.sh
./ci/vhd-build-validate.sh --mode build --output uftc.vhd
```

What it validates:

- output VHD exists
- physical VHD file size passes a minimum-byte gate (default 1 GiB)
- virtual disk size passes a minimum-byte gate when `qemu-img` is available (default 12 GiB)
- container/integrity checks (`file` format hint plus `qemu-img check` when available)

Useful variants:

```bash
# Validate an existing artifact without rebuilding
./ci/vhd-build-validate.sh --mode validate-only --output uftc.vhd

# CI deterministic leg example
./ci/vhd-build-validate.sh --mode build --output uftc.vhd --no-cache
```

## Important post-build step

The generated VHD is not fully finalized until it boots once.

At first login the thin-client launcher checks for `/usr/bin/firstboot` and runs it. That script:

- generates or refreshes initramfs
- installs the XanMod kernel only if it was not already baked into the image
- restores the hostname to `thinclient`
- removes itself
- powers the machine off

Relevant files:

- [tcfiles/thinclient](tcfiles/thinclient)
- [tcfiles/firstboot](tcfiles/firstboot)

This is why the README says to boot the generated image in a VM once before treating it as a golden image.

## Safe test loop

If you want a low-risk way to start modifying the repo, use this loop:

1. Edit the files you care about.
2. Run `./build.sh`.
3. Boot `uftc.vhd` in a VM.
4. Let first boot finish and power off.
5. Boot it again and validate the actual thin-client behavior.

That keeps your feedback cycle focused on one artifact.

## About ISO generation

This repository now includes a minimal installer ISO pipeline via `build-installer-iso.sh`.

It wraps a Clonezilla Live base ISO, injects your built UFTC image payload, and adds an unattended boot entry that:

1. boots the live environment
2. finds the first non-removable disk that is not the boot USB
3. writes UFTC to that disk
4. powers off automatically

## ISO build validation

Use the ISO validation wrapper after VHD validation to build and verify installer media.

```bash
chmod +x ci/iso-build-validate.sh
./ci/iso-build-validate.sh --mode build --input-vhd uftc.vhd --output-iso uftc-installer.iso
```

What it validates:

- output ISO exists and passes a minimum-size gate
- ISO format (`ISO 9660`) when `file` is available
- required payload path (`/uftc/uftc.img.zst`) exists in ISO
- BIOS/UEFI boot assets are present (`/syslinux/isolinux.bin` and `BOOTx64.EFI`)
- compressed payload integrity via `zstd -t`

Useful variant:

```bash
# Validate an existing ISO without rebuilding
./ci/iso-build-validate.sh --mode validate-only --output-iso uftc-installer.iso
```

### Requirements for the installer ISO build

- `xorriso`
- `qemu-img` (provided by `qemu-utils` on Debian/Ubuntu)
- `zstd`
- `curl`
- `lsblk` and `findmnt` (provided by `util-linux` on Debian/Ubuntu and Fedora)

Install the required packages before running `build-installer-iso.sh`:

```bash
# Debian/Ubuntu
sudo apt-get update
sudo apt-get install -y qemu-utils xorriso zstd curl util-linux

# Fedora
sudo dnf install -y qemu-img xorriso zstd curl util-linux
```

### Build commands

```bash
chmod +x build-installer-iso.sh
./build-installer-iso.sh
```

If the script fails with `Permission denied` while editing extracted boot files, update to the latest `build-installer-iso.sh` from this repo version (it now makes extracted files writable before patching boot config).

Default output:

- `uftc-installer.iso`

Optional overrides:

```bash
CLONEZILLA_ISO_URL="https://.../clonezilla-live-amd64.iso" \
INPUT_VHD="/path/to/uftc.vhd" \
OUTPUT_ISO="/path/to/custom-installer.iso" \
WORKDIR="/tmp/uftc-installer" \
./build-installer-iso.sh
```

### Important safety notes

- This installer is intentionally non-interactive.
- It will erase the selected target disk.
- Review and adjust target disk detection in `uftc/install.sh` inside the script if your environment needs a stricter policy.

References used for this implementation:

- Clonezilla Live docs: <https://clonezilla.org/live-doc.php>
- Clonezilla advanced boot parameters (`ocs_live_run`, batch mode): <https://clonezilla.org/show-live-doc-content.php?topic=clonezilla-live/doc/99_Misc>

## Best places to start improving the process

If the current process feels intimidating, the highest-value improvements are probably:

1. Add a `make build` or `just build` wrapper with environment checks.
2. Split "build filesystem" and "package disk image" into separate scripts.
3. Make the first-boot finalization explicit in the build docs and release checklist.
4. Add a VM smoke-test script or written checklist.
5. Add validation tests for target disk selection and unattended ISO boot in VM.

## Quick mental model

Think about the repo in three layers:

1. OS contents: [Dockerfile](Dockerfile) and [tcfiles](tcfiles)
2. Disk-image packaging: [build.sh](build.sh) and [d2vm](d2vm)
3. Deployment media: `build-installer-iso.sh`

Once that split is clear, the project is much less intimidating.
