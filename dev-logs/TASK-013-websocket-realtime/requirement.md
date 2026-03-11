# TASK-013: WebSocket 实时数据推送

## 需求背景
当前前端通过 setInterval 轮询获取数据，体验不够优雅：
- 首页容器列表：每 5 秒轮询 `api.listContainers()`
- 容器详情页资源监控：每 3 秒轮询 `api.getContainerStats(id)`
- 日志流：后端已有 SSE 实现但前端未使用 EventSource

希望改用 WebSocket 保持长连接，服务端主动推送变更数据。

## 改造范围

| 页面 | 当前方式 | 目标 |
|------|---------|------|
| `app/page.tsx` 首页 | 5s 轮询容器列表 | WS 推送容器状态变更 |
| `app/containers/[id]/page.tsx` 详情页 | 3s 轮询 stats | WS 推送资源统计 |

## 验收标准
- 后端新增 WebSocket endpoint，支持 JWT 认证
- 前端连接 WS 后，容器状态变更、资源统计实时推送
- 断线自动重连
- 移除原有 setInterval 轮询
- 日志流保持现有 SSE 方案不变（已经是流式的）
