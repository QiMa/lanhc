#!/usr/bin/env python3
"""tailscale -> lanhc 完全改名工具（幂等，可重复执行，修正版）。

方案：
  - 模块路径: tailscale.com -> lanhc.com（保持单级，不展开成 lanhc.com/lanhc）
  - 品牌词分层大小写替换，保留 Go 导出标识符:
      TAILSCALED -> LANHCD
      TAILSCALE  -> LANHC
      Tailscaled -> Lanhcd
      tailscaled -> lanhcd
      Tailscale  -> Lanhc
      tailscale-ipn -> lanhc-gui
      tailscale  -> lanhc
  - 文件名/目录名中的 tailscale 同步改名
  - k8s 资源组 tailscale.com -> lanhc.com（与模块路径一致，避免 group 含斜杠）

保护项（永不改写）：
  - 外部依赖 tailscale.com/client/tailscale/v2
  - 所有 github.com/tailscale/* 第三方模块
  - go.sum（依赖哈希）
  - 本工具自身
  - 大小写混淆测试样例 TaIlScAlE（有意不匹配任何规则）
"""
import os, re, sys

TARGET = sys.argv[1] if len(sys.argv) > 1 else '/home/dev/src/lanhc'
os.chdir(TARGET)

PROTECT_FILES = {'go.sum'}
PROTECT_SUBDIR_ANY = {'node_modules', '__pycache__'}
PROTECT_NAMES = {'lanhc-rename.py', 'rename-lanhc.py'}

TEXT_EXTS = {
    '.go', '.c', '.h', '.sh', '.bash', '.py', '.json', '.yaml', '.yml', '.md',
    '.ts', '.tsx', '.js', '.jsx', '.txt', '.html', '.htm', '.service', '.socket',
    '.target', '.conf', '.tmpl', '.nix', '.rc', '.defaults', '.openrc', '.init',
    '.admx', '.adml', '.sc', '.cgi', '.rules', '.in', '.desktop', '.xml',
    '.mod', '.sum', '.work', '.gitignore', '.dockerignore', '.editorconfig',
    '.toml', '.ini', '.cfg', '.properties', '.plist', '.pbxproj', '.entitlements',
    '.xcconfig', '.swift', '.m', '.mm', '.hpp', '.proto', '.csv', '.feature',
    '.sql', '.graphql', '.ps1', '.psm1', '.bat', '.cmd', '.rst', '.adoc',
}

V2 = 'tailscale.com/client/tailscale/v2'
GH = 'github.com/tailscale/'
S_V2 = '\x00V2\x00'
S_GH = '\x00GH\x00'


BINARY_EXTS = {'.png', '.gif', '.jpg', '.jpeg', '.ico', '.bmp', '.webp', '.svgz', '.syso',
               '.exe', '.dll', '.so', '.dylib', '.a', '.o', '.zip', '.tar', '.gz', '.tgz',
               '.bz2', '.xz', '.zst', '.pdf', '.woff', '.woff2', '.ttf', '.otf', '.eot',
               '.mp3', '.mp4', '.webm', '.wasm', '.db', '.bin', '.plist.bin', '.icns',
               '.keystore', '.jks', '.p12', '.pfx', '.der'}

def is_text_file(p):
    ext = os.path.splitext(p)[1].lower()
    return ext not in BINARY_EXTS

def walk_files():
    for dirpath, dirnames, filenames in os.walk('.'):
        dirnames[:] = [d for d in dirnames if d not in ('.git', 'node_modules', '__pycache__', '.git')]
        for fn in filenames:
            p = os.path.join(dirpath, fn)
            yield p

def rel(p):
    return p[2:] if p.startswith('./') else p

def is_protected(p):
    r = rel(p)
    if r in PROTECT_FILES:
        return True
    if os.path.basename(r) in PROTECT_NAMES and (r.endswith('lanhc-rename.py') or r.endswith('rename-lanhc.py')):
        return True
    return False

def rewrite(p, fn):
    try:
        with open(p, encoding='utf-8') as f:
            old = f.read()
    except (UnicodeDecodeError, IsADirectoryError):
        return False
    new = fn(old)
    if new != old:
        with open(p, 'w', encoding='utf-8') as f:
            f.write(new)
        return True
    return False

def protect(s):
    return s.replace(V2, S_V2).replace(GH, S_GH)

def unprotect(s):
    return s.replace(S_V2, V2).replace(S_GH, GH)

CAP = 'tailscale.com/cap/'
S_CAP = '\x00CAP\x00'

REPO_OLD = 'github.com/tailscale/tailscale'
REPO_NEW = 'github.com/lanhc/lanhc'

def repo_swap(s):
    """上游仓库自身的 GitHub 引用换成 lanhc 仓库。

    必须在 protect() 之前执行，否则会被 github.com/tailscale/* 保护规则
    拦下，导致出现 github.com/tailscale/lanhc 这种半吊子结果。
    """
    return (s.replace(REPO_OLD, REPO_NEW)
             .replace('tailscale/corp#', 'lanhc/corp#'))

def path_swap(s):
    # module path + web/k8s-group refs: tailscale.com -> lanhc.com
    # 例外: tailscale.com/cap/* 是 tailnet 能力(wire protocol)标识,
    # headscale 用官方 tailcfg 常量校验并下发, 改名会导致 SSH/ACL 能力
    # 协商失败, 因此作为协议常量保留。
    s = protect(repo_swap(s)).replace(CAP, S_CAP)
    s = (s.replace('tailscale.com/', 'lanhc.com/')
          .replace('tailscale.com', 'lanhc.com'))
    return unprotect(s.replace(S_CAP, CAP))

def brand_swap(s):
    s = fix_v2_alias(s)
    s = protect(repo_swap(s)).replace(CAP, S_CAP)
    s = s.replace('TAILSCALED', 'LANHCD')
    s = s.replace('TAILSCALE', 'LANHC')
    s = s.replace('Tailscaled', 'Lanhcd')
    s = s.replace('tailscaled', 'lanhcd')
    s = s.replace('Tailscale', 'Lanhc')
    s = s.replace('tailscale-ipn', 'lanhc-gui')
    s = s.replace('tailscale', 'lanhc')
    return unprotect(s.replace(S_CAP, CAP))

def restore_attribution():
    n = 0
    for p in walk_files():
        if is_protected(p):
            continue
        if not is_text_file(p):
            continue
        def sub(t):
            t = t.replace('Copyright (c) Lanhc Inc & contributors', 'Copyright (c) Tailscale Inc & contributors')
            t = t.replace('Lanhc Inc. All Rights Reserved', 'Tailscale Inc. All Rights Reserved')
            return t
        if rewrite(p, sub):
            n += 1
    print(f'layer-attribution: rewrote {n} files (restore BSD copyright)')

def rename_components():
    """Rename path components (directories and file basenames) containing 'tailscale'."""
    # bottom-up for files, then dirs
    ops = []
    for p in walk_files():
        if is_protected(p):
            continue
        d = os.path.dirname(p)
        b = os.path.basename(p)
        nb = brand_swap(b)
        # uppercase Tailscale file names get folded to Lanhc via brand_swap
        if nb != b:
            ops.append(('file', p, os.path.join(d, nb)))
    # directories, deepest first
    for dirpath, dirnames, filenames in os.walk('.'):
        dirnames[:] = [d for d in dirnames if d != '.git']
    all_dirs = []
    for dirpath, dirnames, filenames in os.walk('.'):
        dirnames[:] = [d for d in dirnames if d != '.git']
        for d in dirnames:
            all_dirs.append(os.path.join(dirpath, d))
    for p in sorted(all_dirs, key=lambda x: x.count(os.sep), reverse=True):
        b = os.path.basename(p)
        nb = brand_swap(b)
        if nb != b:
            ops.append(('dir', p, os.path.join(os.path.dirname(p), nb)))
    # apply
    for kind, src, dst in ops:
        if os.path.exists(src) and not os.path.exists(dst):
            os.rename(src, dst)
    return len(ops)

def layer_paths():
    n = 0
    if rewrite('go.mod', path_swap):
        print('  go.mod rewritten')
        n += 1
    for p in walk_files():
        if is_protected(p):
            continue
        if is_text_file(p):
            if rewrite(p, path_swap):
                n += 1
    print(f'layer-path: rewrote {n} files (tailscale.com -> lanhc.com)')

V2_ALIAS = 'lanhcclient'

def fix_v2_alias(s):
    """对未加别名的外部 v2 客户端导入，补别名 lanhcclient 并同步标识符。

    外部依赖 tailscale.com/client/tailscale/v2 的包名是 tailscale，
    品牌词替换会把 `tailscale.Foo` 误改成 `lanhc.Foo`，这里在替换前
    先把标识符改成别名 lanhcclient，避免编译失败。
    """
    if V2 not in s:
        return s
    t = s.replace(V2, S_V2)
    lines = t.split('\n')
    changed = False
    for i, ln in enumerate(lines):
        if ln.strip() == '"%s"' % S_V2:
            lines[i] = ln.replace('"%s"' % S_V2, '%s "%s"' % (V2_ALIAS, S_V2))
            changed = True
    if not changed:
        return s
    t = '\n'.join(lines)
    t = re.sub(r'tailscale\.(?!com\b)(?=[A-Za-z_])', V2_ALIAS + '.', t)
    return t.replace(S_V2, V2)

def layer_brand():
    n = 0
    for p in walk_files():
        if is_protected(p):
            continue
        if is_text_file(p):
            if rewrite(p, brand_swap):
                n += 1
    print(f'layer-brand: rewrote {n} files (tailscale* -> lanhc*)')

def layer_files():
    n = rename_components()
    print(f'layer-files: renamed {n} path components')

if __name__ == '__main__':
    layer = sys.argv[2] if len(sys.argv) > 2 else 'all'
    if layer == 'paths':
        layer_paths()
    elif layer == 'brand':
        layer_brand()
    elif layer == 'files':
        layer_files()
    elif layer == 'all':
        layer_paths()
        layer_brand()
        layer_files()
        restore_attribution()
    else:
        print('usage: lanhc-rename-new.py TARGET [paths|brand|files|all]')
        sys.exit(2)
