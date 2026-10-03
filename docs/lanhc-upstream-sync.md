# lanhc 上游同步与发布

本文记录 lanhc fork 的日常上游同步与发布流程。核心思路：不直接 `rebase`，
而是在临时目录把上游两个 tag 分别重放成 lanhc fork，只取两份 fork 的差分，
应用到 `lanhc-main`。这样不会重复引入全量改名的巨大 diff。

## 分支与 remote 约定

- `lanhc-main`：本地 fork 主线，定制提交堆在上游 tag 之上。
- `github/main`：GitHub 上的发布线。
- `origin`：本地上游 Tailscale 仓库，路径为 `/home/dev/src/tailscale`，用于工具自检和取上游 tag，不是日常 fetch remote。
- 推送永远使用映射 `lanhc-main:main`，不把 `lanhc-main` 暴露为远端分支名。

当前基线：`v1.102.4`。`lanhc-main` 相对 `v1.102.4` 领先 22 个定制提交。

## 同步步骤

先更新本地上游：

```bash
cd /home/dev/src/tailscale
git fetch --tags
```

回到 lanhc 仓库执行同步，例如升级到 `v1.102.5`：

```bash
cd /home/dev/src/lanhc
tool/lanhc-sync-upstream.sh v1.102.4 v1.102.5
```

也可以让脚本顺手 fetch：

```bash
tool/lanhc-sync-upstream.sh --fetch v1.102.4 v1.102.5
```

脚本会：

1. 从上游导出 `FROM_TAG` 和 `TO_TAG` 两个干净快照。
2. 在两个快照上重放 `tool/lanhc-fork-setup.sh`。
3. 对两份改名快照求 diff，隔离出上游真实变更。
4. 把 diff 应用到当前 `lanhc-main` 工作树。

脚本不 commit、不 push，失败不会弄脏工作区。

## 验证

```bash
go build ./...
go vet ./...
go test ./...
```

涉及 Docker 的构建可用：

```bash
docker build .
```

## 提交与发布

确认 diff 只包含上游该 tag 的实际变更后提交：

```bash
git add -A
git commit -s -m "sync: update to tailscale v1.102.5"
```

提交信息遵循 `docs/commit-messages.md`。如果涉及明确目录，优先写
`VERSION,cmd/containerboot: ...` 这类前缀；纯同步可用 `sync:`。

推送发布线：

```bash
GIT_SSH_COMMAND='ssh -o ConnectTimeout=15 -o StrictHostKeyChecking=accept-new' \
  git push github lanhc-main:main
```

推送后确认 CI 全绿。Windows runner 有时因队列无 runner 而一直 `queued`，
不影响本地构建验证。

## 上游变更较大时的注意事项

`tool/lanhc-fork-setup.sh` 是构建 fork 的唯一入口，包含：

- `tool/lanhc-rename.py all`：全量改名。
- `tool/lanhc-rename.py self-test`：对照上游源码检查线上协议字面量。
- setec vendored 内联。
- 若干改名残留修复。

如果上游新增了协议头、能力 URL 或 DERP/control 交换字面量，`self-test`
可能失败。此时需要先更新 `tool/lanhc-rename.py` 中的
`WIRE_PROTOCOL_RESTORATIONS`，再重跑同步。任何时候都应保留
`tool/lanhc-rename.py self-test` 的 PASS 状态。

## 为什么不用 rebase

`lanhc-main` 上堆叠了 22 个定制提交，其中包含全量改名的超大 diff。直接
`rebase v1.102.5` 会在每个定制提交上重放并解决大量冲突。当前方案只把
`v1.102.4 -> v1.102.5` 的真实变更（如 3 个文件）应用到 `lanhc-main`，
既保持线性历史，也避免无谓冲突。
