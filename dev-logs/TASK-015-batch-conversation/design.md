# TASK-015: 批量对话 & Agent 对话交互 — 设计方案

> 创建时间: 2026-03-12

## 一、整体架构

```
┌─────────────┐     POST /api/containers/{id}/chat      ┌──────────────┐
│  前端对话页  │ ──────────────────────────────────────→ │  Go 后端     │
│  (SSE 流式)  │ ←── SSE: data: {"content":"..."}  ──── │  ChatHandler │
└─────────────┘                                          └──────┬───────┘
                                                                │ docker exec
┌─────────────┐     POST /api/batch/chat                 ┌──────▼───────┐
│  首页多选    │ ──────────────────────────────────────→ │  容器内部     │
│  批量发送    │ ←── SSE: 每个Agent进度 ──────────────── │  curl → API  │
└─────────────┘                                          └──────────────┘
```

通信链路: OpenManage 后端 → `docker exec curl` → 容器内 loopback API

## 二、前置条件：启用 OpenClaw HTTP Endpoint

当前 `openclaw.json` 模板没有启用 `/v1/responses` 端点，需要添加：

```json
// 在 gateway 节点下新增 http 配置
"gateway": {
  "port": {{PORT}},
  "mode": "local",
  "bind": "loopback",
  "http": {
    "endpoints": {
      "responses": { "enabled": true }
    }
  },
  ...
}
```

这样容器内 `curl http://127.0.0.1:{{PORT}}/v1/responses` 就能工作。
bind 保持 loopback 不变，安全性不受影响。

## 三、后端设计

### 3.1 新增 Docker Client 方法

在 `backend/docker/client.go` 中新增 `ExecCommand` 方法：

```go
// ExecCommand 在容器内执行命令并返回输出
func (c *Client) ExecCommand(ctx context.Context, containerID string, cmd []string) (string, error)
```

使用 Docker SDK 的 `ContainerExecCreate` + `ContainerExecAttach` 实现。

### 3.2 新增 OpenClaw 通信层

新建 `backend/openclaw/client.go`：

```go
package openclaw

type Client struct {
    Docker *docker.Client
}

// SendMessage 向指定容器的 Agent 发送消息
// 返回 Agent 的回复文本
func (c *Client) SendMessage(ctx context.Context, containerID string, port int, token string, message string) (string, error)

// GetContainerConfig 从容器的 openclaw.json 中读取 port 和 token
func (c *Client) GetContainerConfig(ctx context.Context, containerID string, mountPrefix string) (port int, token string, err error)
```

`SendMessage` 内部执行：
```bash
docker exec <id> curl -sS http://127.0.0.1:<port>/v1/responses \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"model":"openclaw","input":"<message>"}'
```

`GetContainerConfig` 通过读取宿主机挂载目录下的 `openclaw.json` 文件获取 port 和 token（复用现有的 mount 查找逻辑）。

### 3.3 新增 API 端点

| 方法 | 路径 | 说明 | 响应 |
|------|------|------|------|
| POST | `/api/containers/{id}/chat` | 单个 Agent 发送消息 | SSE 流式 |
| POST | `/api/batch/chat` | 批量发送消息 | SSE 流式 |

#### POST `/api/containers/{id}/chat`

请求体：
```json
{
  "message": "请自我介绍一下"
}
```

响应：SSE 流式推送
```
data: {"status":"sending","message":"正在发送消息..."}
data: {"status":"done","reply":"你好，我是...","sessionId":"xxx"}
data: {"status":"error","message":"连接失败"}
```

后端流程：
1. 读取容器的 openclaw.json 获取 port + token
2. 通过 docker exec curl 发送消息
3. 解析响应，通过 SSE 推送给前端
4. 返回 reply 内容（Agent 回复会自动保存到 conversations 目录）

#### POST `/api/batch/chat`

请求体：
```json
{
  "containerIds": ["abc123", "def456"],
  "message": "请去论坛发一个自我介绍帖子"
}
```

响应：SSE 流式推送（复用 BatchCreate 的事件格式）
```
data: {"agent":"agent-01","index":0,"step":"sending","status":"running","message":"正在发送..."}
data: {"agent":"agent-01","index":0,"step":"done","status":"done","message":"发送成功"}
data: {"agent":"agent-02","index":1,"step":"sending","status":"running","message":"正在发送..."}
data: {"agent":"agent-02","index":1,"step":"error","status":"error","message":"容器未运行"}
data: {"step":"batch-done","status":"done","message":"完成 1/2"}
```

后端流程：
1. 验证所有容器存在且 running
2. 并发（goroutine）向每个容器发送消息
3. 每个容器的进度通过 SSE 实时推送
4. 全部完成后发送 `batch-done` 事件

## 四、前端设计

### 4.1 单个 Agent 对话交互（改造现有对话页面）

改造 `frontend/src/app/containers/[id]/conversations/page.tsx`：

在右侧详情面板底部增加消息输入区域：
```
┌──────────────────────────────────────┐
│  对话消息列表（已有）                  │
│  ...                                  │
│  user: 你好                           │
│  assistant: 你好！我是...             │
│                                       │
├──────────────────────────────────────┤
│  [输入框........................] [发送] │
└──────────────────────────────────────┘
```

- 输入框 + 发送按钮，固定在底部
- 发送后调用 `POST /api/containers/{id}/chat`
- SSE 接收回复，追加到消息列表
- 发送完成后刷新对话列表（左侧会出现新对话）
- 支持「新建对话」按钮（清空当前选中，发送时自动创建新 session）

### 4.2 首页批量发送

改造 `frontend/src/app/page.tsx`：

1. 新增「批量指令」模式切换按钮（在顶部导航栏，与"新建容器"、"批量创建"并列）
2. 点击后进入选择模式：
   - 每张 Agent 卡片左上角出现 checkbox
   - 只有 running 状态的 Agent 可选
   - 底部浮出操作栏：已选 N 个 | [输入指令...] [发送] [取消]
3. 发送后跳转到进度页面（或 modal），复用 batch-create 的 SSE 进度展示模式

### 4.3 前端 API 层新增

在 `frontend/src/lib/api.ts` 中新增：

```typescript
// 单个 Agent 发送消息（返回 SSE URL）
sendMessage: (id: string, message: string) => ...

// 批量发送消息（返回 SSE URL）
batchChat: (containerIds: string[], message: string) => ...
```

## 五、实现步骤

### Phase 1: 基础通信层（后端）
1. 修改 `openclaw.json` 模板，启用 responses endpoint
2. 新增 `docker/client.go` → `ExecCommand` 方法
3. 新建 `backend/openclaw/client.go`，实现 `SendMessage` + `GetContainerConfig`
4. 新增 `POST /api/containers/{id}/chat` handler（SSE）

### Phase 2: 单 Agent 对话交互（前端）
5. 改造对话页面，增加输入框和发送功能
6. 实现 SSE 接收和消息追加
7. 新建对话功能

### Phase 3: 批量发送（前后端）
8. 新增 `POST /api/batch/chat` handler（SSE，并发）
9. 首页增加多选模式和批量操作栏
10. 批量发送进度展示

## 六、风险点

1. **docker exec 超时**: Agent 回复可能很慢（AI 推理），需要设置合理的超时时间（建议 120s）
2. **并发压力**: 批量发送时多个 goroutine 同时 exec，需要控制并发数（建议 semaphore 限制 5 个）
3. **responses endpoint 未启用**: 已有容器的 openclaw.json 不会自动更新，只有新创建的容器才会有。需要提示用户手动更新或提供一键更新功能
