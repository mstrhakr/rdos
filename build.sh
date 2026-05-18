#!/usr/bin/env bash

# BuildKit enables parallel layer execution, cache mounts, and better layer caching.
export DOCKER_BUILDKIT=1

sudo docker build . -t uftc
sudo rm -f uftc.vhd
sudo ./d2vm convert uftc:latest -o uftc.vhd --bootloader grub --boot-size 4000 --size 14G --network-manager none "$@"
