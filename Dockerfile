# syntax=docker/dockerfile:1
FROM debian:trixie

COPY tcfiles/debian.sources /etc/apt/sources.list.d/debian.sources

# Stable system packages — this layer is only invalidated when the list below changes.
# BuildKit cache mounts keep the apt/dpkg cache between builds so packages are not
# re-downloaded on every run; use DOCKER_BUILDKIT=1 (or Docker 23+ default) to benefit.
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get install -y \
        sudo curl wget \
        xterm xinit x11-xserver-utils \
        fvwm yad light \
        freerdp3-x11 \
        wpasupplicant iw rfkill net-tools ethtool \
        systemd-resolved \
        polkitd mingetty \
        pulseaudio pamixer \
        cups mesa-utils \
        firmware-linux firmware-linux-nonfree \
        firmware-iwlwifi firmware-realtek firmware-atheros firmware-brcm80211 \
        open-vm-tools \
        ffmpeg \
        enca nano udiskie mc mtr \
        adwaita-icon-theme-legacy libfuse2 \
        libcap2-bin

# Disable audible terminal/system bell where possible.
RUN printf '%s\n' 'blacklist pcspkr' 'install pcspkr /bin/false' > /etc/modprobe.d/nobeep.conf && \
    printf '%s\n' 'set bell-style none' >> /etc/inputrc

# XanMod kernel — installed during Docker build so no VM first-boot is needed.
# INITRD=No suppresses update-initramfs here (it would fail without a live kernel);
# d2vm runs privileged and regenerates the initramfs correctly during conversion.
# Falls back gracefully if the package is temporarily unavailable from deb.xanmod.org.
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    install -d -m 755 /etc/apt/keyrings && \
    wget -qO - https://dl.xanmod.org/archive.key | gpg --dearmor -o /etc/apt/keyrings/xanmod-archive-keyring.gpg && \
    echo 'deb [signed-by=/etc/apt/keyrings/xanmod-archive-keyring.gpg] http://deb.xanmod.org releases main' > /etc/apt/sources.list.d/xanmod-release.list && \
    apt-get update && \
    INITRD=No DEBIAN_FRONTEND=noninteractive apt-get install -y linux-xanmod-lts-x64v1 || \
        echo 'Warning: linux-xanmod-lts-x64v1 not available; d2vm will install a default kernel.'

# Optional: Citrix ICA client — drop icaclient.deb next to the Dockerfile to include it.
COPY icaclient.deb* /tmp/
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get install -y /tmp/icaclient.deb && rm /tmp/icaclient.deb || true

COPY Moonlight.AppImage* /usr/bin/moonlight

COPY tcfiles/thinclient /usr/bin/thinclient
COPY tcfiles/tc-settings /usr/bin/tc-settings
COPY tcfiles/tc-configure-network /usr/bin/tc-configure-network
COPY tcfiles/tc-configure-wifi /usr/bin/tc-configure-wifi
COPY tcfiles/tc-scan-wifi /usr/bin/tc-scan-wifi
COPY tcfiles/tc-wifi-wizard /usr/bin/tc-wifi-wizard
COPY tcfiles/set-hostname /usr/bin/set-hostname
COPY tcfiles/auto-maintenance.debian /usr/bin/auto-maintenance
COPY tcfiles/099_tc /etc/sudoers.d/099_tc
COPY tcfiles/usb-access.rules /etc/udev/rules.d/usb-access.rules
RUN chown root:root /etc/sudoers.d/099_tc && chmod 440 /etc/sudoers.d/099_tc
RUN chmod +x /usr/bin/*
# Allow mtr to send raw ICMP packets without root (capability survives into the final image)
RUN setcap cap_net_raw+ep /usr/bin/mtr-packet

RUN mkdir -p /etc/systemd/system/getty@tty1.service.d
COPY tcfiles/autologin /etc/systemd/system/getty@tty1.service.d/override.conf
RUN systemctl enable getty@tty1.service

COPY tcfiles/tc-copyconfig.service /etc/systemd/system/tc-copyconfig.service
RUN systemctl enable tc-copyconfig.service

COPY tcfiles/tc-copywpa.service /etc/systemd/system/tc-copywpa.service
RUN systemctl enable tc-copywpa.service

COPY tcfiles/tc-networkconfig.service /etc/systemd/system/tc-networkconfig.service
RUN systemctl enable tc-networkconfig.service

COPY tcfiles/tc-wifipower.service /etc/systemd/system/tc-wifipower.service
RUN systemctl enable tc-wifipower.service

COPY tcfiles/dhcp.network /etc/systemd/network/dhcp.network
COPY tcfiles/systemd-resolved.conf /etc/tmpfiles.d/systemd-resolved.conf
RUN systemctl enable systemd-networkd.service systemd-resolved.service

COPY tcfiles/xorg.conf /etc/X11/xorg.conf.d/thinclient.conf

#This line is for pipewire, because pipewire has limited mic support its currently replaced with pulseaudio
#Pulseaudio has the auto switch behavior by default
#COPY tcfiles/pipewire-pulse.conf /etc/pipewire/pipewire-pulse.conf.d/thinclient.conf

RUN useradd -ms /bin/bash thinclient -G video,audio,netdev,render,cdrom,plugdev

COPY tcfiles/.fvwm /home/thinclient/.fvwm
COPY tcfiles/bashrc /home/thinclient/.bashrc
COPY tcfiles/xinitrc /home/thinclient/.xinitrc
COPY Version /tcversion
COPY tcconfig_override* /home/thinclient/

# Block stock files from being tampered with to harden even more
RUN chown -R root:thinclient /home/thinclient/ && chmod 1775 /home/thinclient/

USER thinclient
WORKDIR /home/thinclient

RUN touch dynamic_hostname

