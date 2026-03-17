# TASK-018: 定时任务执行失败 — 错误报告

> 分析日期: 2026-03-13 | 更新: 2026-03-14

## 问题现象

多个 Agent 的定时任务（Cron + Heartbeat）未正确执行，任务持续静默失败，前端无任何错误提示。

---

## 实际验证结果

### financebot (5aa1eff7c8e9)

| 字段 | 值 |
|------|-----|
| 任务 ID | `f15536ee-5af8-4b55-b243-4a1548ae30f4` |
| Schedule | `*/5 * * * *`（每 5 分钟） |
| Enabled | true |
| 连续错误次数 | **22 次**（3/13）→ **29 次**（3/14，仍在增长） |
| lastRunStatus | error |
| lastDurationMs | 245852（~4 分钟空转） |
| 错误信息 | `Channel is required (no configured channels detected). Set delivery.channel explicitly or use a main session with a previous channel.` |

### message (176ebbc54e77)

| 字段 | 值 |
|------|-----|
| Cron 任务 | 空（`jobs: []`） |
| Heartbeat 配置 | `every: "30m"`（已配置但无法验证是否执行） |
| channels | `{}`（空） |

---

## 根因分析

### 核心问题: `channels: {}` + 默认 `--channel "last"`

两个容器的 `openclaw.json` 中 channels 均为空对象：

```json
"channels": {}
```

OpenClaw cron 执行后需要通过 channel 投递结果。执行链路：

```
openclaw cron add（默认 --channel "last"）
  → 任务触发，执行 Agent turn
  → 尝试通过 "last" channel 投递结果
  → 容器从未建立过任何 channel 连接
  → "last" 解析失败
  → 报错: "Channel is required"
  → 标记为 error，等待下次触发
  → 循环往复，永远失败
```

Heartbeat 同理 — 依赖 channel 投递，`channels: {}` 导致静默失败。

---

## 涉及代码 5 个问题

### 问题 1（P0）: AddCronJob 缺少 `--no-deliver`

`backend/openclaw/client.go:270`:

```go
cmd := []string{"openclaw", "cron", "add", "--name", name, "--cron", schedule,
    "--message", prompt, "--session", "isolated", "--json"}
```

未指定 `--no-deliver`，导致默认使用 `--channel "last"` + announce 模式。纯 API 驱动的容器没有 channel，必须显式禁用投递。

### 问题 2（P0）: openclaw.json 模板 channels 为空

`backend/templates/openclaw.json:109`:

```json
"channels": {}
```

所有通过 OpenManage 创建的容器都继承这个空配置，意味着所有容器的 cron/heartbeat 都会遇到同样的问题。

### 问题 3（P1）: Cron 错误状态未暴露给前端

`backend/openclaw/client.go` 的 `rawCronJob.State` 只解析了时间字段：

```go
State struct {
    NextRunAtMs int64 `json:"nextRunAtMs"`
    LastRunAtMs int64 `json:"lastRunAtMs"`
    // 缺少:
    // LastRunStatus     string `json:"lastRunStatus"`
    // LastError         string `json:"lastError"`
    // ConsecutiveErrors int    `json:"consecutiveErrors"`
}
```

前端 `CronJob` 类型同样缺少错误字段。用户在管理页面看到任务"已启用"、有"上次执行时间"，完全不知道任务已连续失败 29 次。

### 问题 4（P1）: Heartbeat 执行状态不可观测

`GetHeartbeat`/`SetHeartbeat` 只读写 `openclaw.json` 中的配置值，无法获取实际执行状态。前端显示"间隔: 30m"给用户正在工作的错觉，但实际可能从未成功执行过。

### 问题 5（P2）: Cron 默认超时过短

`openclaw cron add` 默认 `--timeout 30000`（30 秒）。Agent 执行论坛交互（检查通知 → 浏览话题 → AI 生成回复 → 发帖）通常需要 60-120 秒。即使 channel 问题修复后，30 秒超时仍会导致大量任务失败。

---

## 修复方案

### P0 — 修复 cron 任务无法执行（阻塞性）

| 修改文件 | 修改内容 |
|----------|---------|
| `backend/openclaw/client.go` AddCronJob | 命令添加 `--no-deliver` 参数 |
| `backend/openclaw/client.go` AddCronJob | 添加 `--timeout-seconds 120` |
| 已有 cron 任务 | 需删除重建，或通过 CLI 手动修复 delivery 配置 |

### P1 — 暴露错误状态到前端

| 修改文件 | 修改内容 |
|----------|---------|
| `backend/openclaw/client.go` rawCronJob | State 增加 LastRunStatus/LastError/ConsecutiveErrors |
| `backend/openclaw/client.go` CronJob | 增加 LastStatus/LastError/ErrorCount 字段 |
| `frontend/src/lib/api.ts` CronJob 类型 | 增加对应字段 |
| `frontend/src/app/containers/[id]/cron/page.tsx` | 错误状态红色标记 + 错误信息展示 |

### P2 — Heartbeat 可观测性 + 超时优化

| 修改文件 | 修改内容 |
|----------|---------|
| `backend/openclaw/client.go` | 研究 openclaw heartbeat 状态查询 CLI |
| `frontend/.../cron/page.tsx` | Heartbeat 区域增加实际执行状态 |
| `backend/templates/openclaw.json` | 评估是否需要默认 channel 配置 |
