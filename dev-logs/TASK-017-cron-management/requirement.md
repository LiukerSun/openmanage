# TASK-017: Agent 定时任务管理（Heartbeat + Cron）

## 需求背景
用户需要在前端查看每个 Agent 的定时任务（heartbeat 和 cron job），查看执行结果，并能创建/修改/启停定时任务。

## OpenClaw 定时任务机制

### Heartbeat
- 配置路径: `openclaw.json` → `agents.defaults.heartbeat`
- 字段: `every`（间隔，如 "30m"）、`mode`（如 "next-heartbeat"）
- 行为由 `workspace/HEARTBEAT.md` 定义
- 禁用: 删除 heartbeat 块

### Cron Jobs
- 通过 CLI 管理: `openclaw cron add/list/run/enable/disable/edit/remove`
- 全局开关: `openclaw.json` → `cron: { enabled: true, maxConcurrentRuns: 2 }`
- 个别 job 定义在运行时状态中，不在 openclaw.json

## 功能需求

### 后端 API
1. `GET /api/containers/{id}/cron` — 获取 cron 列表 + heartbeat 配置
2. `POST /api/containers/{id}/cron` — 创建 cron job
3. `PUT /api/containers/{id}/cron/{jobId}` — 修改 cron job
4. `DELETE /api/containers/{id}/cron/{jobId}` — 删除 cron job
5. `POST /api/containers/{id}/cron/{jobId}/toggle` — 启用/禁用
6. `POST /api/containers/{id}/cron/{jobId}/run` — 立即执行
7. `PUT /api/containers/{id}/heartbeat` — 修改 heartbeat 间隔

### 前端页面
- 容器详情下新增 `/containers/[id]/cron` 页面
- 显示 heartbeat 状态和间隔（可编辑）
- Cron job 列表（名称、schedule、状态、上次执行时间）
- 创建/编辑 cron job 表单
- 启用/禁用/立即执行按钮

## 实现方案

### 后端: 通过 docker exec 执行 openclaw CLI
- `openclaw cron list --json` → 获取 cron 列表
- `openclaw cron add --every "1h" --prompt "..." --session isolated` → 创建
- `openclaw cron enable/disable {id}` → 启停
- `openclaw cron run {id}` → 立即执行
- `openclaw cron remove {id}` → 删除
- Heartbeat: 读写 openclaw.json 的 `agents.defaults.heartbeat` 字段

### 前端: 新增 cron 管理页面
- 路由: `/containers/[id]/cron`
- 复用现有容器详情页布局
