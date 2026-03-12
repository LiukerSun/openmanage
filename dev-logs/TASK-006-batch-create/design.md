# TASK-006: 批量创建 Agent — 设计方案

## 整体思路

复用现有的单个创建逻辑（`ContainerHandler.Create`），后端新增 `POST /api/batch-create` 接口，串行创建每个 Agent，通过 SSE 流式推送每个 Agent 的创建进度。前端新增 `/batch-create` 页面。

## 后端设计

### 新增 Request 类型 (`model/types.go`)

```go
type BatchCreateRequest struct {
    Prefix      string `json:"prefix"`      // 名称前缀，如 "agent"
    Count       int    `json:"count"`       // 创建数量
    StartPort   int    `json:"startPort"`   // 起始端口
    Image       string `json:"image"`       // 镜像（可选，默认 openclaw:latest）
    Description string `json:"description"` // 统一描述（可选，用于 AI 生成）
}
```

### 新增 Handler (`handler/container.go`)

`func (h *ContainerHandler) BatchCreate(w, r)`:
1. 解析 `BatchCreateRequest`
2. 校验：prefix 非空、count 1~20、startPort 合法
3. 开启 SSE 流
4. 循环 count 次，每次：
   - 生成名称：`{prefix}-{i+1}`（如 agent-1, agent-2）
   - 端口：`startPort + i`
   - 数据路径：自动生成 `/home/evan/.openclaw-{name}`
   - 复用现有的 prepare → template → AI → docker 四步逻辑
   - SSE 事件格式：`{agent: "agent-1", index: 0, step: "prepare", status: "running", message: "..."}`
5. 全部完成后发送 `{step: "batch-done", status: "done", total: N, success: M, failed: F}`

关键：单个 Agent 创建失败不中断整个批次，记录失败继续下一个。

### 路由注册 (`main.go`)

```go
r.Post("/api/batch-create", ch.BatchCreate)
```

## 前端设计

### 新增页面 `frontend/src/app/batch-create/page.tsx`

表单字段：
- 名称前缀（必填）
- 创建数量（1~20，默认 5）
- 起始端口（默认 18790）
- Agent 描述（可选，统一描述）

进度展示：
- 每个 Agent 一行，显示名称 + 当前步骤 + 状态图标
- 底部汇总：成功 N 个 / 失败 M 个
- 全部完成后显示"返回首页"按钮

### Dashboard 入口

在首页 `+ 新建容器` 旁边加一个 `批量创建` 按钮。

## 文件变更清单

| 文件 | 变更 |
|------|------|
| `backend/model/types.go` | 新增 `BatchCreateRequest` |
| `backend/handler/container.go` | 新增 `BatchCreate()` 方法 |
| `backend/main.go` | 注册路由 |
| `frontend/src/app/batch-create/page.tsx` | 新增批量创建页面 |
| `frontend/src/app/page.tsx` | 首页加入口按钮 |
