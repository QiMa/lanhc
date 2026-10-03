#!/bin/sh
# install-agent.sh - install lanhc-agent from an unpacked lanhc release tarball.
#
# The tarball contains:
#   lanhc-agent                        agent binary
#   agent/lanhc-agent.service          systemd unit
#   agent/lanhc-agent.defaults         environment file
#
# Usage (run from inside the unpacked tarball directory):
#   sudo TS_AUTHKEY=tskey-auth-XXXX ./install-agent.sh
#
# Environment overrides:
#   LANHC_AGENT_PREFIX       install prefix          (default /usr/local)
#   LANHC_AGENT_BIN_DIR      binary directory        (default $PREFIX/bin)
#   LANHC_AGENT_SYSTEMD_DIR  systemd unit directory  (default /etc/systemd/system)
#   LANHC_AGENT_ENV_DIR      environment directory   (default /etc/default)
#   LANHC_AGENT_STATE_DIR    node state directory    (default /var/lib/lanhc-agent)
#   NO_SYSTEMD=1             only install files, never call systemctl
set -eu

SELF_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

# 脚本和单元在包的 agent/ 子目录里，二进制在包根目录；两种布局都支持，
# 也支持把 agent/ 目录单独拷到别处后再执行。
if [ -f "$SELF_DIR/lanhc-agent" ]; then
    PKG_DIR=$SELF_DIR
elif [ -f "$SELF_DIR/../lanhc-agent" ]; then
    PKG_DIR=$(CDPATH= cd -- "$SELF_DIR/.." && pwd)
else
    echo "install-agent: cannot find lanhc-agent next to the script or one level up" >&2
    exit 1
fi

BIN_SRC="$PKG_DIR/lanhc-agent"
UNIT_SRC="$SELF_DIR/lanhc-agent.service"
DEFAULTS_SRC="$SELF_DIR/lanhc-agent.defaults"

PREFIX=${LANHC_AGENT_PREFIX:-/usr/local}
BIN_DIR=${LANHC_AGENT_BIN_DIR:-$PREFIX/bin}
SYSTEMD_DIR=${LANHC_AGENT_SYSTEMD_DIR:-/etc/systemd/system}
ENV_DIR=${LANHC_AGENT_ENV_DIR:-/etc/default}
STATE_DIR=${LANHC_AGENT_STATE_DIR:-/var/lib/lanhc-agent}

for f in "$BIN_SRC" "$UNIT_SRC" "$DEFAULTS_SRC"; do
    if [ ! -f "$f" ]; then
        echo "install-agent: missing $f" >&2
        echo "run this from the unpacked lanhc release directory" >&2
        exit 1
    fi
done

if [ "$(id -u)" -ne 0 ]; then
    echo "install-agent: must run as root (or set NO_SYSTEMD=1 with writable dirs)" >&2
    exit 1
fi

echo "==> installing lanhc-agent"
mkdir -p "$BIN_DIR" "$SYSTEMD_DIR" "$ENV_DIR" "$STATE_DIR"
install -m 0755 "$BIN_SRC" "$BIN_DIR/lanhc-agent"

echo "==> installing systemd unit and defaults"
install -m 0644 "$UNIT_SRC" "$SYSTEMD_DIR/lanhc-agent.service"
if [ -f "$ENV_DIR/lanhc-agent" ]; then
    touch "$ENV_DIR/lanhc-agent"
else
    install -m 0600 "$DEFAULTS_SRC" "$ENV_DIR/lanhc-agent"
fi

if [ -n "${TS_AUTHKEY:-}" ]; then
    if grep -q '^TS_AUTHKEY=' "$ENV_DIR/lanhc-agent" 2>/dev/null; then
        sed -i "s|^TS_AUTHKEY=.*|TS_AUTHKEY=$TS_AUTHKEY|" "$ENV_DIR/lanhc-agent"
    else
        printf 'TS_AUTHKEY=%s\n' "$TS_AUTHKEY" >> "$ENV_DIR/lanhc-agent"
    fi
fi

if [ "$BIN_DIR" != "/usr/local/bin" ]; then
    sed -i "s|^ExecStart=/usr/local/bin/lanhc-agent|ExecStart=$BIN_DIR/lanhc-agent|" \
        "$SYSTEMD_DIR/lanhc-agent.service"
fi

if [ "${NO_SYSTEMD:-0}" = "1" ] || ! command -v systemctl >/dev/null 2>&1; then
    echo "==> files installed; systemd activation skipped"
    exit 0
fi

echo "==> enabling lanhc-agent.service"
systemctl daemon-reload
systemctl enable --now lanhc-agent.service
systemctl --no-pager --lines=20 status lanhc-agent.service || true
