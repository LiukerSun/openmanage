# TASK-009: 实现记录

## 改动文件

### 后端
- `backend/docker/client.go` — 新增 `RemoveContainer`、`RenameContainer` 方法
- `backend/model/types.go` — 新增 `UpdateContainerRequest`
- `backend/handler/container.go` — 新增 `Delete`、`Update` handler，`ContainerHandler` 增加 `MountPrefix` 字段
- `backend/main.go` — 注册 `DELETE /{id}` 和 `PUT /{id}` 路由，传入 `MountPrefix`

### 前端
- `frontend/src/lib/api.ts` — 新增 `deleteContainer`、`updateContainer`
- `frontend/src/app/page.tsx` — 仪表盘添加删除按钮，action 类型扩展
- `frontend/src/app/containers/[id]/page.tsx` — 详情页添加重命名（内联编辑）和删除功能（带 ConfirmDialog）

## 关键决策
- 删除使用 `Force: true`，无论容器是否运行都直接删除
- 重命名通过 Docker API 的 `ContainerRename` 实现，无需重建容器
