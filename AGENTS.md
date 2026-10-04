# AGENTS.md — lanhc

## 定位
tailscale 的品牌化 fork（模块路径 `lanhc.com`），产出 `lanhc`/`lanhcd`/
`lanhc-agent`/`lanhc-gui.exe`。上游为 `/home/dev/src/tailscale`。

## 约定
- 分支 `lanhc-main`，定制只推 `github` = `git@github.com:QiMa/lanhc.git`；
  `origin` 是本地 tailscale 源，**不要推到上游官方分支**。
- 改名边界见 `tools/lanhc-rename.py`（归档于 lanhc-workspace）：品牌层改名，
  但 HTTP 头/control 子协议/能力 URL 等线上协议字面量保持 tailscale 原值。
- 构建：`make`、`./build_dist.sh`、`./build_docker.sh`；发行包相对 tag 递增
  `+lanhcN`（当前基线 `v1.102.4`）。
- `lanhc-agent` 是设备侧伴随进程（tsnet 入网），不装 Codex；只读采集 + 白名单命令。
- 检查：`make vet` / `make lint` / `make check`。

## 关系
- 客户端与 `headscale`（`/home/dev/src/headscale`）协议互通。
- `lanhc-agent` 数据经 `mcp-lanhc` 供 `ops-runner` 诊断；整体架构见
  `/home/dev/src/lanhc-workspace/docs/PROJECT-RELATIONSHIPS.md`。
