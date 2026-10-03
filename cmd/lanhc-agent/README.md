# lanhc-agent

设备侧伴随进程：跟随 `lanhc` 运行，通过 `lanhc.com/tsnet` 加入同一 tailnet，
只暴露**白名单**只读 HTTP 接口，供控制面的 `ops-runner`/`lanhc-hub` 取证。

- 不装 Codex、不持有 LLM 凭据。
- 不嵌进 `lanhcd`，独立二进制、独立升级。
- 不接受任意 shell：写操作只有模板化命令（`smartctl-long` / `smartctl-info`）。

## 最快上线（发行包）

发布 tarball 里已带 `lanhc-agent` 和 `agent/` 目录。解包后一条命令完成安装和注册：

```bash
tar xzf lanhc_1.102.4+lanhc9_linux_amd64.tar.gz
cd lanhc_1.102.4+lanhc9_linux_amd64
sudo TS_AUTHKEY=tskey-auth-XXXX ./agent/install-agent.sh
```

脚本会：

1. 把 `lanhc-agent` 装到 `/usr/local/bin`。
2. 把 `agent/lanhc-agent.service` 拷到 `/etc/systemd/system`。
3. 把 `agent/lanhc-agent.defaults` 拷到 `/etc/default/lanhc-agent`，并写入 `TS_AUTHKEY`。
4. `systemctl daemon-reload && systemctl enable --now lanhc-agent`。

首次上线只需填 `TS_AUTHKEY`，其余全部用默认值：

- 节点名：`<本机主机名>-agent`
- 控制面：编译进二进制的正式控制面（`ipn.DefaultControlURL`）
- 状态目录：`/var/lib/lanhc-agent`
- 监听地址：`tailnet :8088`
- tags：由 preauthkey 下发（推荐在 headscale 创建带 `--tags tag:lanhc-agent` 的 key）

推荐的 headscale 发 key 命令：

```bash
headscale preauthkeys create --user <username> --tags tag:lanhc-agent --expiration 24h
```

这样 agent 侧保持零 tag 参数，不会出现“preauthkey 已带 tag、客户端又请求 tag”的注册冲突。

已注册节点身份保存在状态目录，之后可把 `/etc/default/lanhc-agent` 里的
`TS_AUTHKEY` 清空，重启不会要求重新注册。

## 手动运行（不装 systemd）

```bash
TS_AUTHKEY=tskey-auth-XXXX lanhc-agent -dir /var/lib/lanhc-agent -listen :8088
```

可选覆盖：

| 环境变量 | 等价参数 | 默认 |
| --- | --- | --- |
| `LANHC_AGENT_HOSTNAME` | `-hostname` | `<本机主机名>-agent` |
| `LANHC_AGENT_DIR` | `-dir` | `/var/lib/lanhc-agent` |
| `LANHC_AGENT_CONTROL_URL` | `-control-url` | 编译期控制面 |
| `LANHC_AGENT_LISTEN` | `-listen` | `:8088` |
| `LANHC_AGENT_TAGS` | `-tags` | 空（tag 由 preauthkey 下发） |
| `TS_AUTHKEY` / `TS_AUTH_KEY` | `-auth-key` | 空（首次注册必填） |

> **preauthkey 与 tags 的规则**：首选在 headscale 用带 tag 的 preauthkey，
> agent 侧 `LANHC_AGENT_TAGS` 保持空。不要在两边同时指定 `tag:lanhc-agent`，
> 否则 headscale 会拒绝注册（`requested tags are invalid or not permitted`）。

## 自检（不接入 tailnet）

```bash
lanhc-agent -selfcheck
```

## API（全部只读，除受控 exec）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/v1/healthz` | 存活 |
| GET | `/v1/inventory` | 主机/OS/内核/CPU/内存/磁盘拓扑 |
| GET | `/v1/health` | load、内存、交换、根分区、failed systemd units |
| GET | `/v1/disk/list` | `smartctl --scan-open` 可寻址物理盘（含 PERC/DELL RAID 成员） |
| GET | `/v1/disk/smart?dev=/dev/sda` | SMART 全量（smartctl -j），可加 `&type=megaraid,N` 指定 `-d` |
| GET | `/v1/logs?scope=dmesg\|journal\|mce\|edac&tail=500` | 受限日志摘要 |
| POST | `/v1/exec` | 仅模板化命令 `smartctl-long` / `smartctl-info` |

## systemd 单元（随包分发）

实际分发版本在 `agent/lanhc-agent.service`，特点是：

- `ExecStart=/usr/local/bin/lanhc-agent`，参数全部来自 `EnvironmentFile` 和编译默认值。
- `EnvironmentFile=-/etc/default/lanhc-agent`。
- `StateDirectory=lanhc-agent` 自动创建 `/var/lib/lanhc-agent`。
- `ProtectSystem=full` / `ProtectHome=true` / `NoNewPrivileges=true`，即使被攻破也只读系统。
- `PartOf=lanhcd.service`，跟随 lanhcd 一起启停。
