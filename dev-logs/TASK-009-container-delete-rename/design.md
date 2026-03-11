# TASK-009: 设计方案

## 后端 API

### DELETE /api/containers/{id}
- `docker/client.go` 新增 `RemoveContainer` 方法，使用 `ContainerRemove(Force: true)` 强制删除
- `handler/container.go` 新增 `Delete` handler

### PUT /api/containers/{id}
- `model/types.go` 新增 `UpdateContainerRequest{Name string}`
- `docker/client.go` 新增 `RenameContainer` 方法，调用 `ContainerRename`
- `handler/container.go` 新增 `Update` handler，解析请求体后调用 rename

### 路由注册
- `main.go` 在 `/api/containers/{id}` 下注册 `DELETE` 和 `PUT`

## 前端

### api.ts
- 新增 `deleteContainer(id)` — DELETE 请求
- 新增 `updateContainer(id, {name})` — PUT 请求

### 仪表盘 page.tsx
- action 类型扩展加入 `delete`
- 按钮区域新增「删除」按钮（bg-red-900），带确认弹窗

### 容器详情页 containers/[id]/page.tsx
- 标题旁「重命名」按钮，点击切换为 input 内联编辑，回车或点保存提交
- 底部「删除容器」按钮，ConfirmDialog 确认后删除并 router.push("/")
