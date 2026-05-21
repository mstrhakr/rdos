# RDOS Build Sudoers Setup

This document describes the static sudoers model for RDOS builds on local development machines and self-hosted runners.

## Overview

Build setup is intentionally split into two separate scripts:

1. `setup-build-deps.sh`: installs host dependencies
2. `setup-build-sudoers.sh`: installs static passwordless sudo rules for build commands

Keeping these steps separate lets operators choose between manual sudo approval during builds or static no-password rules for approved commands.

## Runner / Dev Setup

Run these once per machine:

```bash
bash ./setup-build-deps.sh
./setup-build-sudoers.sh
```

After that, `./build.sh` and `./build.sh --ab` can run without repeated sudo prompts for the allowed build commands.

## What setup-build-sudoers.sh installs

The script writes `/etc/sudoers.d/RDOS-build-nopasswd` and validates it with `visudo` before install.

Rules are static and limited to build pipeline operations:

- `./build.sh` (raw production/recovery image assembly)
- `./build-installer-iso.sh` (installer payload generation)
- `./build-ab-disk.sh` (A/B composition from `--prod-raw/--recovery-raw`)

`build.sh` now self-elevates once (`sudo`) and performs privileged loop/mount
operations in that elevated context. The sudoers policy therefore grants the
top-level build scripts rather than individual helper binaries.

## Removal

To remove the static rule file:

```bash
./setup-build-sudoers.sh --remove
```

## Troubleshooting

### Sudoers install fails validation

Re-run setup and check output:

```bash
./setup-build-sudoers.sh
```

The script validates with `visudo -cf` before writing to `/etc/sudoers.d/RDOS-build-nopasswd`.

### Docker rule missing

If Docker is not installed when you run `setup-build-sudoers.sh`, the script skips the Docker rule. Install dependencies first, then run `setup-build-sudoers.sh` again.

## In-Image Sudoers (Thin Client)

The thin client image (`tcfiles/099_tc`) keeps a separate static policy for runtime administration commands.
