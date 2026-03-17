# Issue Title

Bug: Cron 定时任务因 channel 配置缺失持续执行失败（连续 29 次错误）

# Issue Body

## 问题描述

通过 OpenManage 创建的 Cron 定时任务**全部无法正常执行**，任务持续以 `error` 状态循环触发。以 `financebot` 容器为例，一个每 5 分钟执行的 cron 任务已**连续失败 29 次**，且错误次数仍在增长。

前端管理页面**无任何错误提示**，用户只能看到任务"已启用"和"上次执行时间"，完全不知道任务一直在失败。

## 错误信息

```
Channel is required (no configured channels detected).
Set delivery.channel explicitly or use a main session with a previous channel.
```

## 复现步骤

1. 通过 OpenManage 创建一个 OpenClaw 容器
2. 在定时任务管理页面创建一个 Cron 任务（任意 schedule + prompt）
3. 等待任务触发
4. 进入容器执行 `openclaw cron list --json` 查看实际状态
5. 观察到 `lastRunStatus: "error"`，`consecutiveErrors` 持续增长

## 根因分析

### 核心问题：`channels: {}` + 默认 `--channel "last"`

`backend/templates/openclaw.json` 中 channels 配置为空对象：

```json
"channels": {}
```

`backend/openclaw/client.go` 的 `AddCronJob` 方法：

```go
cmd := []string{"openclaw", "cron", "add", "--name", name, "--cron", schedule,
    "--message", prompt, "--session", "isolated", "--json"}
```

未指定 `--no-deliver`，OpenClaw 默认使用 `--channel "last"` + announce 模式投递结果。但容器从未建立过任何 channel 连接，`"last"` 解析失败，导致每次执行都报错。

### 执行链路

```
openclaw cron add（默认 --channel "last"）
  → 任务触发，执行 Agent turn
  → 尝试通过 "last" channel 投递结果
  → 容器无任何已配置 channel
  → 报错 "Channel is required"
  → 标记 error，等待下次触发
  → 循环往复，永远失败
```

### Heartbeat 同样受影响

Heartbeat 也依赖 channel 投递，`channels: {}` 导致 heartbeat 静默失败。前端显示"间隔: 30m"但实际可能从未成功执行。

## 涉及的 5 个子问题

### 1. [P0] AddCronJob 缺少 `--no-deliver` 参数
- 文件：`backend/openclaw/client.go:270`
- 纯 API 驱动的容器没有 channel，必须显式禁用投递

### 2. [P0] openclaw.json 模板 channels 为空
- 文件：`backend/templates/openclaw.json:109`
- 所有通过 OpenManage 创建的容器都继承空配置

### 3. [P1] Cron 错误状态未暴露给前端
- 文件：`backend/openclaw/client.go` rawCronJob.State
- 缺少 `lastRunStatus`、`lastError`、`consecutiveErrors` 字段解析
- 前端 `CronJob` 类型和 cron 页面均无错误展示

### 4. [P1] Heartbeat 执行状态不可观测
- `GetHeartbeat`/`SetHeartbeat` 只读写配置值，无法获取实际执行状态

### 5. [P2] Cron 默认超时过短
- `openclaw cron add` 默认 30 秒超时
- Agent 论坛交互任务通常需要 60-120 秒

## 实际验证数据

**financebot 容器** (`openclaw cron list --json` 输出)：

```json
{
  "state": {
    "lastRunStatus": "error",
    "lastStatus": "error",
    "consecutiveErrors": 29,
    "lastDurationMs": 245852,
    "lastError": "Channel is required (no configured channels detected)..."
  },
  "delivery": {
    "mode": "announce",
    "channel": "last"
  }
}
```

**message 容器**：cron 任务列表为空，heartbeat 配置 `every: "30m"` 但 channels 同样为空。

## 建议修复方案

### P0 — 修复 cron 无法执行

```go
// backend/openclaw/client.go AddCronJob
cmd := []string{"openclaw", "cron", "add",
    "--name", name, "--cron", schedule,
    "--message", prompt, "--session", "isolated",
    "--no-deliver", "--timeout-seconds", "120",
    "--json"}
```

### P1 — 暴露错误状态

- `rawCronJob.State` 增加 `LastRunStatus`/`LastError`/`ConsecutiveErrors`
- `CronJob` 增加对应前端字段
- 前端 cron 页面显示错误状态（红色标记 + 错误信息）

### P2 — 超时与可观测性

- 默认超时改为 120 秒
- 研究 heartbeat 状态查询能力

## 环境信息

- OpenClaw 版本：2026.3.8 (3caab92)
- OpenManage 分支：master
- 容器镜像：fourplayers/openclaw:latest
