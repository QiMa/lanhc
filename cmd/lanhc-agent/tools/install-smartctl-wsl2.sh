#!/bin/sh
# Install the WSL2 canary smartctl shim without sudo.
#
#   ./install-smartctl-wsl2.sh [~/bin]
#
# This builds a local lanhc/smartmontools:7.4 image, copies
# smartctl-wsl2.sh into the target dir as `smartctl`, and prints the PATH
# snippet needed by the lanhc-agent service. Physical Linux hosts should use
# distro smartmontools instead.
set -eu

here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
dest=${1:-"$HOME/bin"}
mkdir -p "$dest"
cp "$here/smartctl-wsl2.sh" "$dest/smartctl"
chmod +x "$dest/smartctl"
docker build -f "$here/smartmontools-wsl2.Dockerfile" -t lanhc/smartmontools:7.4 "$here"
echo "installed: $dest/smartctl"
echo "ensure PATH contains $dest, then restart lanhc-agent"
