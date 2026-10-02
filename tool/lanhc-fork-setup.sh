#!/bin/bash
# 把上游 tailscale 源码树完整转换为 lanhc fork。
#
# 用法: lanhc-fork-setup.sh <源码目录>
#
# 幂等：可对同一目录重复执行。
#
# 步骤:
#   1. 全量改名 tailscale -> lanhc（模块路径 / 品牌词 / 文件与目录名）
#      + 还原与 headscale/官方组件交换的线上协议字面量，并用上游源码自检
#   2. vendored setec 客户端（其源码引用 tailscale.com/atomicfile 与
#      tailscale.com/types/logger，改名后必须指向本仓库）
#   3. 修正改名带来的测试与常量残留
#   4. 把本脚本与改名工具复制进 <源码目录>/tool/
set -euo pipefail

TARGET=${1:?用法: lanhc-fork-setup.sh <源码目录> [上游源码目录]}
UPSTREAM_DIR=${2:-${TAILSCALE_UPSTREAM_DIR:-/home/dev/src/tailscale}}
SRC=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
TARGET=$(cd "$TARGET" && pwd)

python3 "$SRC/lanhc-rename.py" "$TARGET" all
python3 "$SRC/lanhc-rename.py" "$TARGET" self-test "$UPSTREAM_DIR"

cd "$TARGET"

# --- vendored setec ---------------------------------------------------------
SETEC_VER=$(awk '$1=="github.com/tailscale/setec" {print $2; exit}' go.mod)
if [ -z "$SETEC_VER" ]; then
  echo "go.mod 中找不到 github.com/tailscale/setec" >&2
  exit 1
fi
SETEC="$(go env GOMODCACHE)/github.com/tailscale/setec@${SETEC_VER}"
if [ ! -d "$SETEC" ]; then
  echo "缺少 setec 模块缓存: $SETEC" >&2
  exit 1
fi

mkdir -p third_party/setec/client/setec third_party/setec/types/api
cp "$SETEC"/client/setec/{cache.go,client.go,fields.go,fileclient.go,store.go,watcher.go} third_party/setec/client/setec/
cp "$SETEC"/types/api/api.go third_party/setec/types/api/
cp "$SETEC"/LICENSE third_party/setec/LICENSE
cat > third_party/setec/go.mod <<'GOMOD'
module github.com/tailscale/setec

go 1.24.0
GOMOD
sed -i 's#tailscale\.com/atomicfile#lanhc.com/atomicfile#g; s#tailscale\.com/types/logger#lanhc.com/types/logger#g' third_party/setec/client/setec/*.go
if ! grep -q 'replace github.com/tailscale/setec' go.mod; then
  printf '\nreplace github.com/tailscale/setec => ./third_party/setec\n' >> go.mod
fi

# --- 改名残留修正 ----------------------------------------------------------
python3 - <<'PYEOF'
import pathlib, uuid

# exename.go: 两个候选二进制名改名后重复，收敛为单一判断
f = pathlib.Path('version/exename.go')
s = f.read_text()
s = s.replace('return preppedExeName == "lanhc-gui" || preppedExeName == "lanhc-gui"',
              'return preppedExeName == "lanhc-gui"')
f.write_text(s)

# version_internal_test.go: 有意保留的大小写混淆样例不会被替换，
# 其期望值为小写后的原始字面量。
f = pathlib.Path('version/version_internal_test.go')
s = f.read_text()
s = s.replace('"TaIlScAlE-iPn.ExE",\n\t\t\t"lanhc-gui",',
              '"TaIlScAlE-iPn.ExE",\n\t\t\t"tailscale-ipn",')
f.write_text(s)

# tsweb 测试: 大小写混淆样例改为与 lanhc.com 匹配的 LaNhC.CoM
f = pathlib.Path('tsweb/tsweb_test.go')
s = f.read_text()
s = s.replace('"http://TaIlScAlE.CoM/spongebob", lanhcHost, "http://TaIlScAlE.CoM/spongebob", false',
              '"http://LaNhC.CoM/spongebob", lanhcHost, "http://LaNhC.CoM/spongebob", false')
f.write_text(s)

# version/mkversion 的 Windows MSI 产品码由 pkgs URL 派生，改名后需重算
f = pathlib.Path('version/mkversion/mkversion_test.go')
s = f.read_text()
for arch in ('amd64', 'arm64', 'x86'):
    old_uuid = str(uuid.uuid5(uuid.NAMESPACE_URL,
        f'https://pkgs.tailscale.com/unstable/tailscale-setup-1.15.129-{arch}.msi')).upper()
    new_uuid = str(uuid.uuid5(uuid.NAMESPACE_URL,
        f'https://pkgs.lanhc.com/unstable/lanhc-setup-1.15.129-{arch}.msi')).upper()
    s = s.replace(old_uuid, new_uuid)
f.write_text(s)
PYEOF

# --- 把工具复制进仓库 ------------------------------------------------------
mkdir -p tool
cp "$SRC/lanhc-rename.py" tool/lanhc-rename.py
cp "$SRC/lanhc-fork-setup.sh" tool/lanhc-fork-setup.sh
chmod +x tool/lanhc-fork-setup.sh

echo "lanhc fork 已就绪: $TARGET"
