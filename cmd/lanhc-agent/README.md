# lanhc-agent

设备侧伴随进程：跟随 `lanhc` 运行，通过 `lanhc.com/tsnet` 加入同一 tailnet，
只暴露**白名单**只读 HTTP 接口，供控制面的 `ops-runner`/`lanhc-hub` 取证。

- 不装 Codex、不持有 LLM 凭据。
- 不嵌进 `lanhcd`，独立二进制、独立升级。
- 不接受任意 shell：写操作只有模板化命令（`smartctl-long` / `smartctl-info`）。

## 运行

```bash
lanhc-agent \
  -hostname r930-01-agent \
  -dir /var/lib/lanhc-agent \
  -control-url https://headscale.lanhc.com \
  -auth-key <一次性 preauthkey> \
  -tags tag:lanhc-agent \
  -listen :8088
```

- `-auth-key` 仅首次注册需要；已注册节点状态存在 `-dir` 下，之后可省略。
- `-listen` 的端口只落在 tailnet 网卡，不暴露在物理网口；ACL 应只允许 `tag:ai-ops-runner` 访问。

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
| GET | `/v1/disk/smart?dev=/dev/sda` | SMART 全量（smartctl -j） |
| GET | `/v1/logs?scope=dmesg\|journal\|mce\|edac&tail=500` | 受限日志摘要 |
| POST | `/v1/exec` | 仅模板化命令 `smartctl-long` / `smartctl-info` |

## systemd 单元（示例）

```ini
[Unit]
Description=lanhc-agent
After=network-online.target lanhcd.service

[Service]
ExecStart=/usr/local/bin/lanhc-agent -dir /var/lib/lanhc-agent -listen :8088
Restart=on-failure
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```
