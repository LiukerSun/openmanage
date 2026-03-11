# TASK-010: 实现记录

## 改动文件
- `backend/docker/client.go` — `CreateContainer` 方法

## 修复方案
在 `ContainerStart` 失败时，立即调用 `ContainerRemove(Force: true)` 清理刚创建的容器，并返回空 ID 和错误。

### 修改前
```go
if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
    return resp.ID, err
}
```

### 修改后
```go
if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
    c.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
    return "", err
}
```

## 根因分析
Docker 的 `ContainerCreate` 和 `ContainerStart` 是两步操作。Create 成功后名字就被占用了，即使 Start 失败（端口冲突等），名字仍然被占着。需要在 Start 失败时回滚 Create。
