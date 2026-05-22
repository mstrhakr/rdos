# syntax=docker/dockerfile:1
FROM debian:trixie

ARG XANMOD_ARCHIVE_SHA256=ed26eb39330fd296cd037b8229adccea0197b21989ec0a1ad4f4f74f5a41c7a7
ARG XANMOD_ARCHIVE_FINGERPRINT=D38D7D1DA1349567ADED882D86F7D09EE734E623

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
        wpasupplicant iw rfkill net-tools ethtool wireguard-tools \
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
    libcap2-bin \
    grub-common

# Disable audible terminal/system bell where possible.
RUN printf '%s\n' 'blacklist pcspkr' 'install pcspkr /bin/false' > /etc/modprobe.d/nobeep.conf && \
    printf '%s\n' 'set bell-style none' >> /etc/inputrc

# XanMod kernel — installed during Docker build so no VM first-boot is needed.
# We regenerate initramfs in-image and disable resume probing to avoid boot delays
# from stale/non-existent swap UUIDs in cloned environments.
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    install -d -m 755 /etc/apt/keyrings && \
    wget -qO /tmp/xanmod-archive.key https://dl.xanmod.org/archive.key && \
    echo "$XANMOD_ARCHIVE_SHA256  /tmp/xanmod-archive.key" | sha256sum -c - && \
    test "$(gpg --show-keys --with-colons /tmp/xanmod-archive.key | awk -F: '$1=="fpr"{print $10; exit}')" = "$XANMOD_ARCHIVE_FINGERPRINT" && \
    gpg --dearmor -o /etc/apt/keyrings/xanmod-archive-keyring.gpg /tmp/xanmod-archive.key && \
    rm -f /tmp/xanmod-archive.key && \
    echo 'deb [signed-by=/etc/apt/keyrings/xanmod-archive-keyring.gpg] http://deb.xanmod.org releases main' > /etc/apt/sources.list.d/xanmod-release.list && \
    apt-get update && \
    INITRD=No DEBIAN_FRONTEND=noninteractive apt-get install -y linux-xanmod-lts-x64v1 || \
        echo 'Warning: linux-xanmod-lts-x64v1 not available; continuing with current kernel packages in image.' && \
    rm -f /etc/apt/sources.list.d/xanmod-release.list && \
    printf 'RESUME=none\n' > /etc/initramfs-tools/conf.d/resume && \
    if ls /lib/modules/* >/dev/null 2>&1; then update-initramfs -u -k all; fi

# Optional: Citrix ICA client and Moonlight AppImage from the build context.
RUN --mount=type=bind,source=.,target=/build-context,ro \
    --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    ICA_DEB="$(find /build-context -maxdepth 1 -type f -name 'icaclient.deb*' | head -n1)" && \
    if [ -n "$ICA_DEB" ]; then \
        cp "$ICA_DEB" /tmp/icaclient.deb && \
        apt-get install -y /tmp/icaclient.deb && \
        rm -f /tmp/icaclient.deb; \
    else \
        echo 'Optional icaclient.deb not found; skipping Citrix package install.'; \
    fi && \
    MOONLIGHT_BIN="$(find /build-context -maxdepth 1 -type f -name 'Moonlight.AppImage*' | head -n1)" && \
    if [ -n "$MOONLIGHT_BIN" ]; then \
        install -m 0755 "$MOONLIGHT_BIN" /usr/bin/moonlight; \
    else \
        echo 'Optional Moonlight.AppImage not found; skipping moonlight install.'; \
    fi

COPY tcfiles/thinclient /usr/bin/thinclient
COPY tcfiles/tc-settings /usr/bin/tc-settings
COPY tcfiles/tc-import-usb /usr/bin/tc-import-usb
COPY tcfiles/tc-configure-network /usr/bin/tc-configure-network
COPY tcfiles/tc-configure-wifi /usr/bin/tc-configure-wifi
COPY tcfiles/tc-configure-wireguard /usr/bin/tc-configure-wireguard
COPY tcfiles/tc-scan-wifi /usr/bin/tc-scan-wifi
COPY tcfiles/tc-wifi-wizard /usr/bin/tc-wifi-wizard
COPY tcfiles/set-hostname /usr/bin/set-hostname
COPY tcfiles/auto-maintenance.debian /usr/bin/auto-maintenance
COPY tcfiles/tc-ota-updater /usr/bin/tc-ota-updater
COPY tcfiles/tc-ota-configure-timer /usr/bin/tc-ota-configure-timer
COPY tcfiles/tc-health-check /usr/bin/tc-health-check
COPY tcfiles/tc-ota-rollback /usr/bin/tc-ota-rollback
COPY tcfiles/099_tc /etc/sudoers.d/099_tc
COPY tcfiles/usb-access.rules /etc/udev/rules.d/usb-access.rules
RUN chown root:root /etc/sudoers.d/099_tc && chmod 440 /etc/sudoers.d/099_tc
RUN chmod +x \
    /usr/bin/thinclient \
    /usr/bin/tc-settings \
    /usr/bin/tc-import-usb \
    /usr/bin/tc-configure-network \
    /usr/bin/tc-configure-wifi \
    /usr/bin/tc-configure-wireguard \
    /usr/bin/tc-scan-wifi \
    /usr/bin/tc-wifi-wizard \
    /usr/bin/set-hostname \
        /usr/bin/auto-maintenance \
        /usr/bin/tc-ota-updater \
        /usr/bin/tc-ota-configure-timer \
        /usr/bin/tc-health-check \
        /usr/bin/tc-ota-rollback
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

COPY tcfiles/tc-ota-updater.service /etc/systemd/system/tc-ota-updater.service
COPY tcfiles/tc-ota-updater.timer /etc/systemd/system/tc-ota-updater.timer
RUN systemctl enable tc-ota-updater.timer

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
COPY tcfiles/profile /home/thinclient/.profile
COPY tcfiles/xinitrc /home/thinclient/.xinitrc
COPY Version /tcversion

# Block stock files from being tampered with to harden even more
RUN chown -R root:thinclient /home/thinclient/ && chmod 1775 /home/thinclient/

USER thinclient
WORKDIR /home/thinclient

RUN touch dynamic_hostname

