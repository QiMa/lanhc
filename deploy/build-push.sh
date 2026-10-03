#!/usr/bin/env bash
# lanhc（自研客户端 + lanhc-agent）固定版本号发布流程。
#
# 用法：
#   ./deploy/build-push.sh                     # 自动算出下一个 +lanhcN
#   ./deploy/build-push.sh 1.102.5+lanhc11     # 显式指定版本
#
# 步骤：
#   1. 调用本仓库 deploy/build-tailscale-custom.sh 构建发行包
#   2. 上传到发行目录 /lucky/lanhc-hugo/lanhc/（对外 https://lanhc.com/lanhc/）
#   3. 归档旧版、刷新稳定别名、重算 SHA256SUMS 并校验
#
# 环境变量：
#   --no-upload         只构建，不上传（本地验收/CI 冒烟）
#   TARGETS             默认 linux/amd64 linux/arm64（加 windows/amd64 windows/arm64 可覆盖）
#   CONTROL_URL         默认 https://headscale.lanhc.com
#   ADMIN_URL           默认 https://console.lanhc.com
#   LANHC_RELEASE_HOST  默认 lucky@lanhc.com
#   LANHC_RELEASE_DIR   默认 /lucky/lanhc-hugo/lanhc

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

BASE_VERSION="$(sed 's/^v//' "$REPO_ROOT/VERSION.txt" | tr -d '[:space:]')"
OUT="${OUT:-/home/dev/out/lanhc}"
OUT_SCAN="${OUT_SCAN:-/home/dev/out}"

NO_UPLOAD=0
VERSION=""
for arg in "$@"; do
  case "$arg" in
    --no-upload) NO_UPLOAD=1 ;;
    -*) echo "未知参数: $arg" >&2; exit 2 ;;
    *) VERSION="$arg" ;;
  esac
done
if [ -z "$VERSION" ]; then
  last="$(ls "$OUT_SCAN"/lanhc*/lanhc_"${BASE_VERSION}"+lanhc*_linux_amd64.tar.gz "$OUT"/lanhc_"${BASE_VERSION}"+lanhc*_linux_amd64.tar.gz 2>/dev/null | sed 's/.*+lanhc//; s/_linux_amd64.*//' | sort -n | tail -1 || true)"
  n=0
  [ -n "$last" ] && n="$last"
  VERSION="${BASE_VERSION}+lanhc$((n + 1))"
fi

echo "==> 构建版本 ${VERSION}"
OUT="$OUT" \
VERSION="$VERSION" \
REPO="$REPO_ROOT" \
TARGETS="${TARGETS:-linux/amd64 linux/arm64}" \
CONTROL_URL="${CONTROL_URL:-https://headscale.lanhc.com}" \
ADMIN_URL="${ADMIN_URL:-https://console.lanhc.com}" \
bash "$REPO_ROOT/deploy/build-tailscale-custom.sh"

if [ "$NO_UPLOAD" = "1" ]; then
  echo "==> --no-upload：跳过上传与归档，产物在 $OUT"
  echo "✅ 完成（仅本地）: ${VERSION}"
  exit 0
fi

HOST="${LANHC_RELEASE_HOST:-lucky@lanhc.com}"
DIR="${LANHC_RELEASE_DIR:-/lucky/lanhc-hugo/lanhc}"
echo "==> 上传到 ${HOST}:${DIR}"
scp "$OUT"/lanhc_"${VERSION}"_* "$HOST":"$DIR"/

echo "==> 归档旧版、刷新稳定别名、校验 SHA256SUMS"
ssh "$HOST" VERSION="$VERSION" BASE_VERSION="$BASE_VERSION" DIR="$DIR" bash -s <<'REMOTE'
set -e
cd "$DIR"

# 归档除当前版本以外的所有带版本号包（稳定别名保留）
for f in lanhc_[0-9]*+lanhc*_linux_amd64.tar.gz \
         lanhc_[0-9]*+lanhc*_linux_arm64.tar.gz \
         lanhc_[0-9]*+lanhc*_windows_amd64.zip \
         lanhc_[0-9]*+lanhc*_windows_arm64.zip; do
  [ -f "$f" ] || continue
  case "$f" in
    lanhc_${VERSION}_*) ;;
    *) mv -f "$f" archive/ 2>/dev/null || true ;;
  esac
done

for f in lanhc_${VERSION}_linux_amd64.tar.gz lanhc_${VERSION}_linux_arm64.tar.gz; do
  [ -f "$f" ] && cp -f "$f" "${f#_${VERSION}}"
done
for f in lanhc_${VERSION}_windows_amd64.zip lanhc_${VERSION}_windows_arm64.zip; do
  [ -f "$f" ] && cp -f "$f" "${f#_${VERSION}}"
done

printf '%s\n' "$BASE_VERSION" > tailscale-version.txt
sha256sum lanhc_${VERSION}_* lanhc_linux_* lanhc_windows_* tailscale-version.txt > SHA256SUMS
chown lucky:lucky SHA256SUMS tailscale-version.txt lanhc_${VERSION}_* lanhc_linux_* lanhc_windows_* 2>/dev/null || true
sha256sum -c SHA256SUMS
REMOTE

echo "✅ 完成: ${VERSION} 已发布到 https://lanhc.com/lanhc/"
