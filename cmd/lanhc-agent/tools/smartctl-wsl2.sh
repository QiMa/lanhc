#!/bin/sh
# smartctl shim for a WSL2 canary host.
#
# The canary agent must run unprivileged, but WSL2 exposes /dev/sd* as
# "Msft Virtual Disk" behind an emulated SCSI stack that needs raw-IO access,
# and the distro does not ship smartmontools. This shim preserves the exact
# `smartctl <fixed args>` contract lanhc-agent invokes:
#
#   1. Use the real smartctl when present (normal physical Linux hosts).
#   2. Otherwise tunnel through a privileged one-shot container that sees /dev
#      so `smartctl --scan-open` can enumerate block devices.
#
# WSL2 virtual disks do not surface real ATA SMART attributes. On a physical
# Linux host the first branch returns those attributes natively, which is what
# SMART telemetry and the second-hand-drive alert rules actually need.
REAL=/usr/sbin/smartctl
[ -x "$REAL" ] || REAL=/sbin/smartctl
if [ -x "$REAL" ]; then
    exec "$REAL" "$@"
fi

IMG=${LANHC_SMARTCTL_IMAGE:-lanhc/smartmontools:7.4}
if command -v docker >/dev/null 2>&1 && docker image inspect "$IMG" >/dev/null 2>&1; then
    exec docker run --rm --privileged -v /dev:/dev "$IMG" "$@"
fi

echo "smartctl unavailable: install smartmontools or build $IMG" >&2
exit 127
