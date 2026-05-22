# Building RDOS

This repository builds:

- `rdos.vhd` (bootable virtual disk image)
- `rdos-installer.iso` (minimal unattended installer ISO)

The normal flow is still to build `rdos.vhd` first, then optionally package it into the installer ISO.

## What the build actually does

The current build flow is small but it depends on Linux container behavior:

1. `docker build . -t RDOS`
2. `./d2vm convert RDOS:latest -o rdos.vhd --bootloader grub --boot-size 4000 --size 14G --network-manager none`

In practice that means:

- [Dockerfile](Dockerfile) assembles a Debian-based thin-client filesystem.
- [build.sh](build.sh) builds the container and converts it into `rdos.vhd`.
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

Before local builds or runner provisioning, install dependencies first:

```bash
bash ./setup-build-deps.sh
```

From Windows, use `setup-build-deps.cmd` to invoke the same helper through WSL.

Then install static passwordless sudo rules (optional, but useful for CI and repeated local builds):

```bash
./setup-build-sudoers.sh
```

Keeping dependencies and sudoers setup separate lets you choose whether to enter sudo manually during builds or allow only the static build commands without prompts.

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

This creates `/etc/sudoers.d/rdos-build-nopasswd` for the current Linux user and allows passwordless sudo only for static build commands (`docker build`, this repo's `d2vm convert`, and `build-ab-disk.sh`).

To remove the rule later:

```bash
./setup-build-sudoers.sh --remove
```

If the build succeeds, you should get:

- `rdos.vhd`

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
- `iso`: validates structural checks and requires A/B assembly plus ISO-path tooling
- `all`: validates structural checks and requires VHD, A/B assembly, and ISO tooling

CI should call the same script in `ci` mode and choose a path scope per job matrix leg.

## VHD build validation

Use the canonical VHD validation wrapper to run `build.sh` and assert artifact shape.

```bash
chmod +x ci/vhd-build-validate.sh
./ci/vhd-build-validate.sh --mode build --output rdos.vhd
```

What it validates:

- output VHD exists
- physical VHD file size passes a minimum-byte gate (default 1 GiB)
- virtual disk size passes a minimum-byte gate when `qemu-img` is available (default 12 GiB)
- container/integrity checks (`file` format hint plus `qemu-img check` when available)

Useful variants:

```bash
# Validate an existing artifact without rebuilding
./ci/vhd-build-validate.sh --mode validate-only --output rdos.vhd

# CI deterministic leg example
./ci/vhd-build-validate.sh --mode build --output rdos.vhd --no-cache
```

## Post-build behavior

There is no legacy firstboot completion phase in the current raw-image pipeline.
The installed image should boot directly into normal thin-client flow.

## Safe test loop

If you want a low-risk way to start modifying the repo, use this loop:

1. Edit the files you care about.
2. Run `./build.sh`.
3. Boot the built image artifact in a VM.
4. Validate the actual thin-client behavior.

That keeps your feedback cycle focused on one artifact.

## About ISO generation

This repository now includes a minimal installer ISO pipeline via `build-installer-iso.sh`.

It wraps a Clonezilla Live base ISO, injects your built RDOS image payload, and adds an unattended boot entry that:

1. boots the live environment
2. finds the first non-removable disk that is not the boot USB
3. writes RDOS to that disk
4. powers off automatically

## ISO build validation

Use the ISO validation wrapper after VHD validation to build and verify installer media.

```bash
chmod +x ci/iso-build-validate.sh
./ci/iso-build-validate.sh --mode build --input-vhd rdos.vhd --output-iso rdos-installer.iso
```

What it validates:

- output ISO exists and passes a minimum-size gate
- ISO format (`ISO 9660`) when `file` is available
- required payload path (`/RDOS/rdos.img.zst`) exists in ISO
- BIOS/UEFI boot assets are present (`/syslinux/isolinux.bin` and the bundled EFI boot image `efi.img`)
- compressed payload integrity via `zstd -t`

Useful variant:

```bash
# Validate an existing ISO without rebuilding
./ci/iso-build-validate.sh --mode validate-only --output-iso rdos-installer.iso
```

## Orchestrated pipeline

If you want one command that represents your full flow, use the pipeline runner:

```bash
chmod +x ci/pipeline.sh
./ci/pipeline.sh all
```

Step model:

- `pretest`
- `build-img`
- `img-test`
- `build-iso`
- `iso-test`

Dependency behavior:

- running a late step with defaults runs prior steps automatically
- use `--no-deps` to run only one step

Examples:

```bash
# Run through ISO test with dependencies (same as full flow)
./ci/pipeline.sh iso-test

# Run only image validation
./ci/pipeline.sh img-test --no-deps --output-vhd rdos.vhd

# Override shellcheck threshold for pretest
./ci/pipeline.sh pretest --mode ci --shellcheck-severity error
```

## GitHub Actions CI

The repository includes a workflow at `.github/workflows/ci.yml` with two jobs:

- `pretest` on `ubuntu-latest` (hosted)
- `build-and-validate` on a self-hosted Linux runner

The self-hosted job expects runner labels:

- `self-hosted`
- `linux`
- `rdos-builder`

Runner requirements for `build-and-validate`:

- Docker
- shellcheck
- sgdisk (`gdisk` on Debian/Ubuntu)
- mkfs.fat (`dosfstools` on Debian/Ubuntu)
- mkfs.ext4 (`e2fsprogs` on Debian/Ubuntu)
- rsync
- grub-install / grub-editenv (`grub-pc-bin`, `grub-efi-amd64-bin`, `grub-common` on Debian/Ubuntu)
- util-linux (`losetup`, `lsblk`, `findmnt`, `blkid`)
- kpartx
- partprobe (`parted` on Debian/Ubuntu and Fedora)
- xorriso
- qemu-img (`qemu-utils`)
- zstd
- curl

Self-hosted job behavior:

- runs `bash ./ci/pipeline.sh all --mode ci --shellcheck-severity error`
- uploads `rdos.vhd` and `rdos-installer.iso` as artifacts
- skips untrusted fork PRs by default

If your runner uses different labels, update `runs-on` in `.github/workflows/ci.yml`.

## GitHub Actions Release Workflow

The repository includes a release automation workflow at `.github/workflows/release.yml` that:

1. Triggers on every push to `master` (or manually via `workflow_dispatch`)
2. Detects version from the `Version` file (major.minor format, e.g., `2.3`)
3. Auto-increments patch number if `Version` file hasn't changed since last tag
4. Creates a git tag (e.g., `v2.3.0`, `v2.3.1`)
5. Builds full pipeline on self-hosted runner
6. Creates a GitHub Release with artifacts

### Version file format

The `Version` file contains only major.minor (e.g., `2.3`). Patch number is managed automatically.

### Release numbering logic

- If `Version` file changed: reset patch to 0 (e.g., `2.3.0` → `2.4.0`)
- If `Version` file unchanged: increment patch (e.g., `2.3.0` → `2.3.1`)
- First release ever: starts at `v{major}.{minor}.0`
- Stable release numbering only considers stable tags (`vX.Y.Z`) and ignores RC tags (`vX.Y.Z-rc.N`)

Examples:

```text
Scenario 1: Version 2.3 → 2.4
Last tag: v2.3.5
Version file changed: 2.3 → 2.4
Next tag: v2.4.0 (patch reset)

Scenario 2: Version unchanged at 2.3
Last tag: v2.3.5
Version file unchanged: still 2.3
Next tag: v2.3.6 (patch incremented)
```

### Release artifacts

Each release includes:

- `rdos.vhd` (14GB bootable disk image)
- `rdos-installer.iso` (Clonezilla-based installer)

### Triggering a release manually

```bash
git push origin master
# or workflow_dispatch from GitHub Actions UI
```

The workflow will automatically:

1. Detect or increment the version
2. Build via the self-hosted runner
3. Create the release tag and GitHub Release

## Beta / RC Release Workflow

Pre-release builds are created automatically on pull requests to `master`.

### RC Release Requirements

When creating a PR:

1. **Version file must be at least one minor version newer than master**
   - Master is at `2.3` → PR must be `2.4` or higher
   - This ensures RC versions are always newer than stable

2. **PR must contain a valid Version bump** compared to master (see rule 1)

### RC Release Numbering

Each PR push creates an incremented RC version:

- First PR push with version `2.4`: `v2.4.0-rc.1`
- Second push (Version unchanged): `v2.4.0-rc.2`
- Third push: `v2.4.0-rc.3`
- etc.

If you bump Version to `2.5` in the PR, next RC starts at `v2.5.0-rc.1`.

The RC workflow runs on each PR update (`synchronize`), so pushes that do not touch `Version` still produce incremented RC builds while the PR version remains valid.

### RC Release Artifacts

RC releases include versioned artifacts:

- `rdos-2.4.0-rc.1.vhd` (14GB bootable disk image)
- `rdos-installer-2.4.0-rc.1.iso` (Clonezilla-based installer)

RC releases are marked as **pre-release** on GitHub and do not show up in stable release listings.

### Merge to Stable Release

When the PR is merged to `master`:

1. The Version file (e.g., `2.4`) is now on master
2. Next push to master automatically triggers `release.yml`
3. A stable release is created: `v2.4.0`
4. Artifacts are named: `rdos-2.4.0.vhd`, `rdos-installer-2.4.0.iso`
5. Subsequent master pushes (without Version changes) increment: `v2.4.1`, `v2.4.2`, etc.

## Installer ISO Build Requirements

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
sudo apt-get install -y docker.io shellcheck qemu-utils xorriso zstd curl util-linux parted gdisk dosfstools rsync grub-pc-bin grub-efi-amd64-bin grub-common e2fsprogs file

# Fedora
sudo dnf install -y moby-engine ShellCheck qemu-img xorriso zstd curl util-linux parted gdisk dosfstools rsync grub2-pc-modules grub2-efi-x64-modules grub2-tools-extra e2fsprogs file
```

Or use the helper:

```bash
bash ./setup-build-deps.sh
```

### Build commands

```bash
chmod +x build-installer-iso.sh
./build-installer-iso.sh
```

If the script fails with `Permission denied` while editing extracted boot files, update to the latest `build-installer-iso.sh` from this repo version (it now makes extracted files writable before patching boot config).

Default output:

- `rdos-installer.iso`

Optional overrides:

```bash
CLONEZILLA_ISO_URL="https://.../clonezilla-live-amd64.iso" \
INPUT_VHD="/path/to/rdos.vhd" \
OUTPUT_ISO="/path/to/custom-installer.iso" \
WORKDIR="/tmp/rdos-installer" \
./build-installer-iso.sh
```

### Important safety notes

- This installer is intentionally non-interactive.
- It will erase the selected target disk.
- Review and adjust target disk detection in `RDOS/install.sh` inside the script if your environment needs a stricter policy.

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
