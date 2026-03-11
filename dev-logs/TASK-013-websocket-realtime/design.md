# TASK-013: WebSocket 实时推送 - 设计方案

## 技术选型

后端使用 `gorilla/websocket`（Go 生态最成熟的 WS 库），前端使用原生 WebSocket API。

## WebSocket 协议设计

### 连接端点
```
GET /api/ws?token=<jwt>
```
通过 query param 传递 JWT（WebSocket 不支持自定义 header）。

### 消息格式（JSON）

服务端 → 客户端：
```json
{
  "type": "containers",        // 消息类型
  "data": [...]                // 容器列表（完整）
}
```
```json
{
  "type": "container_stats",
  "containerId": "abc123",
  "data": { "cpuPercent": 12.5, ... }
}
```

### 客户端 → 服务端（订阅机制）
```json
{
  "action": "subscribe_stats",
  "containerId": "abc123"
}
```
```json
{
  "action": "unsubscribe_stats",
  "containerId": "abc123"
}
```

客户端连接后自动接收容器列表推送，需要 stats 时主动订阅特定容器。

## 后端实现

### 新增文件：`backend/handler/ws.go`

核心逻辑：
1. 升级 HTTP → WebSocket，验证 JWT
2. 维护一个 Hub（连接管理器），管理所有活跃连接
3. 后台 goroutine 每 3 秒通过 Docker SDK 获取容器列表，有变化时广播
4. 对订阅了 stats 的连接，每 3 秒推送对应容器的资源统计
5. 连接断开时自动清理订阅

### 路由注册
在 `main.go` 中添加：
```go
r.Get("/api/ws", wsHandler.Handle)  // 不需要 RequireAuth 中间件，WS 内部验证
```

## 前端实现

### 新增文件：`frontend/src/lib/ws.ts`

WebSocket 管理 hook：
- `useWebSocket()` — 全局 WS 连接管理
- 自动重连（指数退避，1s → 2s → 4s → 最大 30s）
- 认证失败时跳转登录页

### 修改文件

1. `app/page.tsx` — 移除 setInterval，监听 WS `containers` 消息
2. `app/containers/[id]/page.tsx` — 移除 stats 轮询，通过 WS 订阅/取消订阅 stats

## 改动文件清单

| 文件 | 操作 |
|------|------|
| `backend/go.mod` | 添加 gorilla/websocket 依赖 |
| `backend/handler/ws.go` | 新建，WS handler + Hub |
| `backend/main.go` | 注册 WS 路由 |
| `frontend/src/lib/ws.ts` | 新建，WS 客户端 hook |
| `frontend/src/app/page.tsx` | 改用 WS 接收容器列表 |
| `frontend/src/app/containers/[id]/page.tsx` | 改用 WS 接收 stats |
