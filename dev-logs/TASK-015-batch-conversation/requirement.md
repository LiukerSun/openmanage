# TASK-015: 批量对话 & Agent 对话交互功能

> 创建时间: 2026-03-12

## 需求背景

用户需要能够同时给选中的多个 Agent 创建新对话并发送指令，同时需要一个完整的对话交互界面来查看对话历史和发送消息。

## 功能拆分

### 功能 A: Agent 对话交互（单个 Agent）

在容器详情页的对话功能中，增加完整的对话交互能力：

1. 查看对话列表（已有）
2. 查看对话详情/历史消息（已有）
3. **新增**: 在对话窗口中发送消息，实时展示 Agent 回复
4. **新增**: 创建新对话（新 session）

### 功能 B: 批量发送指令（多个 Agent）

在首页或专门页面，选中多个 Agent，统一发送一条指令：

1. 勾选多个 running 状态的 Agent
2. 输入要发送的指令文本
3. 点击发送，为每个 Agent 创建新对话并发送指令
4. SSE 流式展示每个 Agent 的发送进度/状态
5. 发送完成后可跳转到各 Agent 的对话页面查看回复

## 技术方案

### 通信方式: Docker Exec

通过 `docker exec` 在容器内部执行 curl 命令调用 loopback API，无需修改 openclaw.json 模板。

```bash
docker exec <container_id> curl -sS http://127.0.0.1:<port>/v1/responses \
  -H 'Authorization: Bearer <gateway_token>' \
  -H 'Content-Type: application/json' \
  -d '{"model":"openclaw","input":"<message>"}'
```

### 关键前置条件

1. **获取 Gateway Token**: 需要从容器内的 openclaw.json 中读取 `gateway.auth.token`
2. **获取 Gateway Port**: 需要从容器内的 openclaw.json 中读取 `gateway.port`
3. **启用 HTTP Endpoint**: 当前模板未启用 `/v1/responses`，需要在 openclaw.json 模板中添加 `gateway.http.endpoints.responses.enabled: true`
4. **容器必须 running**: 只有运行中的容器才能执行 exec

### 后端新增 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/containers/{id}/conversations` | 创建新对话并发送消息 |
| POST | `/api/containers/{id}/conversations/{sid}/messages` | 在已有对话中发送消息 |
| POST | `/api/batch/conversations` | 批量给多个 Agent 创建对话并发送指令 |

### 前端新增页面/组件

1. 对话详情页增加消息输入框和发送按钮
2. 首页 Agent 列表增加多选 checkbox
3. 批量发送指令的 modal 或独立页面

## 验收标准

- [ ] 能在单个 Agent 的对话页面发送消息并看到回复
- [ ] 能创建新对话
- [ ] 能在首页选中多个 Agent 批量发送指令
- [ ] 批量发送有进度展示
- [ ] 发送失败有错误提示
