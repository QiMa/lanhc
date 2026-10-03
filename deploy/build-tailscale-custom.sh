#!/usr/bin/env bash
# 构建 lanhc 发行包。
#
# 两种模式：
#   fork 模式（默认，推荐）: 直接从 /home/dev/src/lanhc（自研 fork，已含
#       阶段 1 隔离层 + 阶段 2 改名）构建，不再打补丁。产物为
#       lanhc / lanhcd / lanhc-gui.exe。
#   patch 模式（回滚用）: FORK=0 时，从上游 tag 拉临时 worktree 打
#       tailscale-lanhc.patch，产物为 tailscale / tailscaled / tailscale-ipn.exe。
#
# 定制点：
#   tailscale.com/ipn.DefaultControlURL  默认控制面 => 等价于隐式 --login-server
#   tailscale.com/ipn.DefaultAdminURL    管理台地址 => console.lanhc.com
#
# 产物（linux 为 tar.gz，windows 为 zip）：
#   tailscale-lanhc_<ver>_<os>_<arch>.<ext>
#   tailscale-lanhc_<os>_<arch>.<ext>          # 稳定别名，供安装脚本固定引用
#   tailscale-version.txt                      # 本次构建对应的上游版本号（记录用）
#   SHA256SUMS
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

FORK=${FORK:-1}
if [ "$FORK" = "1" ]; then
    REPO=${REPO:-$REPO_ROOT}
    MODULE=${MODULE:-lanhc.com}
    BIN_PREFIX=${BIN_PREFIX:-lanhc}
    BIN_DAEMON=${BIN_DAEMON:-lanhcd}
    BIN_GUI=${BIN_GUI:-lanhc-gui}
    CMD_CLI=${CMD_CLI:-./cmd/lanhc}
    CMD_DAEMON=${CMD_DAEMON:-./cmd/lanhcd}
    CMD_GUI=${CMD_GUI:-./cmd/systray}
    CMD_AGENT=${CMD_AGENT:-./cmd/lanhc-agent}
    BIN_AGENT=${BIN_AGENT:-lanhc-agent}
    SUFFIX=${SUFFIX:-lanhc}
else
    REPO=${REPO:-$REPO_ROOT/../tailscale}
    MODULE=${MODULE:-tailscale.com}
    BIN_PREFIX=${BIN_PREFIX:-tailscale}
    BIN_DAEMON=${BIN_DAEMON:-tailscaled}
    BIN_GUI=${BIN_GUI:-tailscale-ipn}
    CMD_CLI=${CMD_CLI:-./cmd/tailscale}
    CMD_DAEMON=${CMD_DAEMON:-./cmd/tailscaled}
    CMD_GUI=${CMD_GUI:-./cmd/systray}
    CMD_AGENT=${CMD_AGENT:-}
    BIN_AGENT=${BIN_AGENT:-}
    SUFFIX=${SUFFIX:-tailscale-lanhc}
fi
PKG_NAME=${PKG_NAME:-$SUFFIX}
SVC_DAEMON=${SVC_DAEMON:-$BIN_DAEMON}
SVC_PREFIX=${SVC_PREFIX:-$BIN_PREFIX}
PATCH=${PATCH:-$REPO_ROOT/../tailscale-lanhc.patch}
OUT=${OUT:-/home/dev/out/$SUFFIX}
CONTROL_URL=${CONTROL_URL:-https://headscale.lanhc.com}
ADMIN_URL=${ADMIN_URL:-https://console.lanhc.com}
PROXY_URL=${PROXY_URL:-$CONTROL_URL}
REF=${REF:-v1.102.4}
VERSION=${VERSION:-${REF#v}+lanhc1}

# UPSTREAM_VERSION 是本次构建所基于的上游发行版号，写进 tailscale-version.txt
# 便于追溯（安装脚本已不再据此下载官方 MSI：发行包自带托盘程序）。
# 定制包的版本号可以带 +lanhcN 后缀，这里只留纯数字上游版本。
UPSTREAM_VERSION=${UPSTREAM_VERSION:-}

export GOPROXY=${GOPROXY:-https://goproxy.cn,direct}
export CGO_ENABLED=0
export GOFLAGS=${GOFLAGS:-}

# 与官方 Tailscale 隔离：
#   lanhc_isolated             编译期开关，把官方端点字面量整段编译掉（见
#                              internal/lanhc/），保证 strings 扫描无残留
#   ts_omit_clientupdate       关闭官方自动更新（pkgs.tailscale.com）
#   ts_omit_logtail            关闭官方日志上传（log.tailscale.com）
#   ts_omit_oauthkey           关闭 OAuth 换 authkey（api.tailscale.com）
#   ts_omit_flashappliance     关闭 flash/PVE 镜像下载（pkgs.tailscale.com）
#   ts_omit_aws                关闭 AWS 参数仓库 authkey 解析
#   ts_omit_identityfederation 关闭 workload identity federation
BUILD_TAGS=${BUILD_TAGS:-lanhc_isolated,ts_omit_clientupdate,ts_omit_logtail,ts_omit_oauthkey,ts_omit_flashappliance,ts_omit_aws,ts_omit_identityfederation}

# 所有与 lanhc 控制面相关的链接期替换。值可被环境变量覆盖：
#   LANHC_CONTROL_URL  控制面（等价于 DefaultControlURL / DefaultServerURL）
#   LANHC_ADMIN_URL    管理台
#   LANHC_PROXY_URL    系统代理探测端点（默认 = 控制面）
# 后三项必须在 tailscale-lanhc.patch 里先声明为 var。
CONTROL_URL=${CONTROL_URL:-https://headscale.lanhc.com}
ADMIN_URL=${ADMIN_URL:-https://console.lanhc.com}
PROXY_URL=${PROXY_URL:-$CONTROL_URL}

mkdir -p "$OUT"
work=$(mktemp -d)
# 直接 rm -rf 会留下 git worktree 元数据（git worktree list 里变成 prunable），
# 所以先用 worktree remove 反注册，再删目录。
cleanup() {
    if [ -d "$work/src" ]; then
        git -C "$REPO" worktree remove --force "$work/src" >/dev/null 2>&1 || true
    fi
    rm -rf "$work"
}
trap cleanup EXIT

if [ "$FORK" = "1" ]; then
    echo "==> fork 模式：直接使用 $REPO（不打补丁）"
    if [ -n "$(git -C "$REPO" status --porcelain)" ]; then
        echo "WARNING: $REPO 工作区不干净，将按当前内容构建" >&2
        git -C "$REPO" status --short | head >&2
    fi
    mkdir -p "$work/src"
    # 用 rsync 复制工作区（含未提交改动），排除 .git 与构建缓存
    rsync -a --exclude '.git' --exclude 'third_party' "$REPO"/ "$work/src/" >/dev/null
    rsync -a "$REPO/third_party" "$work/src/" >/dev/null 2>&1 || true
    cd "$work/src"
else
    echo "==> patch 模式：$REPO @ $REF + $PATCH"
    git -C "$REPO" worktree add "$work/src" "$REF" >/dev/null
    cd "$work/src"
    git apply "$PATCH"
fi

# 版本 stamp：默认用发行版本号；构建源自哪个提交也要写进二进制。
VERSION_SHORT=${VERSION_SHORT:-${VERSION%%+*}}
if [ "$FORK" = "1" ]; then
    SRC_COMMIT=$(git -C "$REPO" rev-parse HEAD 2>/dev/null || true)
else
    SRC_COMMIT=$(git -C "$REPO" rev-parse "$REF" 2>/dev/null || true)
fi
version_ldflags="-X ${MODULE}/version.longStamp=${VERSION} -X ${MODULE}/version.shortStamp=${VERSION_SHORT}"
if [ -n "${SRC_COMMIT}" ]; then
    version_ldflags="${version_ldflags} -X ${MODULE}/version.gitCommitStamp=${SRC_COMMIT}"
fi

ldflags="-X ${MODULE}/ipn.DefaultControlURL=${CONTROL_URL} -X ${MODULE}/ipn.DefaultAdminURL=${ADMIN_URL} ${version_ldflags}"
# LoginEndpointForProxyDetermination：系统代理探测端点。
# logtail.DefaultHost：日志上传主机，置空后 logtail 不再连官方。
isolation_ldflags="-X ${MODULE}/net/netmon.LoginEndpointForProxyDetermination=${PROXY_URL}/ -X ${MODULE}/logtail.DefaultHost="

for required_tag in lanhc_isolated ts_omit_clientupdate ts_omit_logtail ts_omit_oauthkey; do
    case ",$BUILD_TAGS," in
        *",${required_tag},"*) ;;
        *) echo "WARNING: BUILD_TAGS 缺少隔离 tag ${required_tag}" >&2 ;;
    esac
done

build() {
    goos=$1; goarch=$2; ext=""
    [ "$goos" = "windows" ] && ext=".exe"
    echo "==> ${goos}/${goarch}  control=${CONTROL_URL}  admin=${ADMIN_URL}"
    mkdir -p "$work/${goos}-${goarch}"
    GOOS=$goos GOARCH=$goarch go build -trimpath -tags "$BUILD_TAGS" \
        -ldflags "$ldflags $isolation_ldflags" \
        -o "$work/${goos}-${goarch}/${BIN_DAEMON}${ext}" ${CMD_DAEMON}
    GOOS=$goos GOARCH=$goarch go build -trimpath -tags "$BUILD_TAGS" \
        -ldflags "$ldflags $isolation_ldflags" \
        -o "$work/${goos}-${goarch}/${BIN_PREFIX}${ext}"  ${CMD_CLI}

    # 设备侧只读取证伴随进程。只在 fork 模式构建：上游没有这个命令。
    if [ -n "${CMD_AGENT}" ]; then
        GOOS=$goos GOARCH=$goarch go build -trimpath -tags "$BUILD_TAGS" \
            -ldflags "$ldflags $isolation_ldflags" \
            -o "$work/${goos}-${goarch}/${BIN_AGENT}${ext}" ${CMD_AGENT}
    fi

    # Windows 托盘程序（阶段 3：自研发行物，不再依赖官方 MSI 的
    # tailscale-ipn.exe）。用上游 cmd/systray，同样走隔离构建。
    # -H windowsgui 让托盘程序不弹控制台窗口（与官方发行方式一致）。
    if [ "$goos" = "windows" ]; then
        GOOS=$goos GOARCH=$goarch go build -trimpath -tags "$BUILD_TAGS" \
            -ldflags "$ldflags $isolation_ldflags -H=windowsgui" \
            -o "$work/${goos}-${goarch}/${BIN_GUI}${ext}" ${CMD_GUI}
    fi
}

targets="${TARGETS:-linux/amd64 linux/arm64 windows/amd64 windows/arm64}"
for t in $targets; do build "${t%%/*}" "${t##*/}"; done

# 取 wintun.dll（tailscaled 在 Windows 上必需，发行包需自带）。
# 优先用本地缓存，缓存在 $OUT/.cache/wintun-<ver>/，避免每次重新下载。
prepare_wintun() {
    ver=${WINTUN_VERSION:-0.14.1}
    cache="$OUT/.cache/wintun-$ver"
    if [ -f "$cache/amd64/wintun.dll" ] && [ -f "$cache/arm64/wintun.dll" ]; then
        echo "$cache"
        return 0
    fi
    mkdir -p "$cache"
    zip="$cache/wintun.zip"
    url=${WINTUN_URL:-https://www.wintun.net/builds/wintun-$ver.zip}
    echo "==> 下载 wintun $ver" >&2
    curl -fsSL --retry 3 -o "$zip" "$url" >&2
    python3 - "$zip" "$cache" <<'PY'
import os, sys, zipfile
src, dst = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(src) as z:
    for n in z.namelist():
        if not n.lower().endswith("wintun.dll"):
            continue
        arch = n.split("/")[-2]
        out = os.path.join(dst, arch)
        os.makedirs(out, exist_ok=True)
        with open(os.path.join(out, "wintun.dll"), "wb") as f:
            f.write(z.read(n))
PY
    echo "$cache"
}

pack() {
    goos=$1; goarch=$2
    d="$work/${goos}-${goarch}"
    if [ "$goos" = "windows" ]; then
        wintun_dir=$(prepare_wintun)
        cp -f "$wintun_dir/$goarch/wintun.dll" "$d/wintun.dll"
        python3 - "$d" "$OUT/${PKG_NAME}_${VERSION#v}_${goos}_${goarch}.zip" \
                "${BIN_PREFIX}.exe" "${BIN_DAEMON}.exe" "${BIN_GUI}.exe" ${BIN_AGENT:+"${BIN_AGENT}.exe"} <<'PY'
import os, sys, zipfile
src, dst, *names = sys.argv[1:]
with zipfile.ZipFile(dst, "w", zipfile.ZIP_DEFLATED) as z:
    for name in list(names) + ["wintun.dll"]:
        z.write(os.path.join(src, name), name)
PY
    else
        # 带上 systemd 单元，安装脚本直接用，不依赖上游 install.sh
        mkdir -p "$d/systemd"
        for f in "${SVC_DAEMON}.service" "${SVC_DAEMON}.defaults" \
                 "${SVC_PREFIX}-online.target" "${SVC_PREFIX}-wait-online.service"; do
            cp -f "./cmd/${CMD_DAEMON#./cmd/}/$f" "$d/systemd/$f"
        done
        # lanhc-agent 的 systemd 单元随包分发，安装脚本直接拷进 /etc/systemd/system。
        if [ -n "${CMD_AGENT}" ]; then
            mkdir -p "$d/agent"
            cp -f "$REPO/cmd/lanhc-agent/lanhc-agent.service" \
                  "$REPO/cmd/lanhc-agent/lanhc-agent.defaults" \
                  "$REPO/cmd/lanhc-agent/install-agent.sh" "$d/agent/" 2>/dev/null || true
        fi
        tar -C "$d" -czf "$OUT/${PKG_NAME}_${VERSION#v}_${goos}_${goarch}.tar.gz" \
            "${BIN_PREFIX}" "${BIN_DAEMON}" ${BIN_AGENT:+"${BIN_AGENT}"} systemd \
            ${CMD_AGENT:+"agent"}
    fi
}

for t in $targets; do pack "${t%%/*}" "${t##*/}"; done

# 稳定别名，安装脚本按固定名字取包，不需要知道版本号。
cd "$OUT"

# 记录本次构建所基于的上游纯数字版本号（不带 +lanhcN 后缀），便于追溯。
if [ -z "$UPSTREAM_VERSION" ]; then
    if [ -f "$work/src/VERSION.txt" ] && grep -Eq '^v?[0-9]+\.[0-9]+\.[0-9]+$' "$work/src/VERSION.txt"; then
        UPSTREAM_VERSION=$(sed 's/^v//' "$work/src/VERSION.txt")
    elif printf '%s' "$REF" | grep -Eq '^v?[0-9]+\.[0-9]+\.[0-9]+$'; then
        UPSTREAM_VERSION=$(printf '%s' "$REF" | sed 's/^v//')
    else
        UPSTREAM_VERSION=${VERSION%%+*}
    fi
fi
printf '%s\n' "$UPSTREAM_VERSION" > tailscale-version.txt
for f in "${PKG_NAME}_${VERSION#v}"_*; do
    alias=$(python3 - "$f" "${VERSION#v}" <<'PY'
import sys
f, ver = sys.argv[1], sys.argv[2]
print(f.replace("_" + ver + "_", "_"))
PY
)
    [ "$f" != "$alias" ] && cp -f "$f" "$alias"
done
# 只收录本次构建的版本 + 稳定别名，避免把历史产物混进清单。
sha256sum "${PKG_NAME}_"*"${VERSION#v}"_* "${PKG_NAME}_"[a-z]*_* > SHA256SUMS

# 隔离扫描：构建产物里不应再出现官方网络端点。控制面、登录、DERP、日志、
# 自动更新这些域名都必须消失。
#
# 唯一允许的 tailscale.com 残留是 Go 模块路径（tailscale.com/<pkg>/...，
# 反射/调试用的类型名，不是可拨号端点）；扫描只认 URL 裸域、带子域名的官方
# 主机和 tailscale.io。
scan_artifacts() {
    local file bad=0 hits
    echo "==> 隔离字符串扫描"
    for file in "$work"/*/"${BIN_PREFIX}" "$work"/*/"${BIN_DAEMON}" \
                "$work"/*/"${BIN_PREFIX}.exe" "$work"/*/"${BIN_DAEMON}.exe" "$work"/*/"${BIN_GUI}.exe"; do
        [ -f "$file" ] || continue
        # 只匹配"能拨号"的官方端点形态：
        #   1) 带子域名的官方主机： controlplane./pkgs./login./log./api./derpN.tailscale.com
        #   2) 任何 tailscale.io（只有官方日志用的 tailnode.log.tailscale.io）
        # 不匹配裸域 https://tailscale.com/kb|/s|/cap/...（文档/协议 URI，
        # 不是主动拨号端点，阶段 2 品牌改名时统一处理），也不匹配 Go 模块
        # 路径 tailscale.com/<pkg>/...（反射/调试类型名）。
        #
        # grep 在"无匹配"时返回 1，会触发 set -e 直接退出；加 || true 才能
        # 让空结果落到下面的 if [ -n ] 判断里。
        hits=$(strings -a "$file" \
            | grep -E '[A-Za-z0-9_-]\.tailscale\.(com|io)|tailscale\.io' \
            | sort -u || true)
        if [ -n "$hits" ]; then
            echo "  FAIL: $file"
            printf '%s\n' "$hits"
            bad=1
        else
            echo "  ok: $(basename "$(dirname "$file")")/$(basename "$file")"
        fi
    done
    if [ "$bad" -ne 0 ]; then
        echo "隔离扫描失败：产物仍包含官方域名，拒绝继续打包。" >&2
        return 1
    fi
    echo "扫描通过：无官方域名残留"
}

# Windows 包内容门禁：安装脚本按固定四个文件解包，缺一个都会装不上。
check_windows_zip() {
    local bad=0 zipf
    echo "==> Windows 包内容检查"
    for zipf in "$OUT"/"${PKG_NAME}"_windows_*.zip; do
        [ -f "$zipf" ] || continue
        if ! python3 - "$zipf" "${BIN_PREFIX}.exe" "${BIN_DAEMON}.exe" "${BIN_GUI}.exe" <<'PY'
import sys, zipfile
want = set(sys.argv[2:]) | {"wintun.dll"}
with zipfile.ZipFile(sys.argv[1]) as z:
    got = set(z.namelist())
missing = want - got
extra = got - want
if missing or extra:
    print("  FAIL: %s missing=%s extra=%s" % (sys.argv[1], sorted(missing), sorted(extra)))
    sys.exit(1)
print("  ok: %s -> %s" % (sys.argv[1], sorted(got)))
PY
        then
            bad=1
        fi
    done
    if [ "$bad" -ne 0 ]; then
        echo "Windows 包内容检查失败：缺文件或有多余文件，拒绝打包。" >&2
        return 1
    fi
}

# 隔离扫描 + Windows 包内容检查作为打包门禁。
scan_artifacts
check_windows_zip

ls -la
