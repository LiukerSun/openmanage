# TASK-003: Discourse API 凭证管理

## 需求背景

Agent 要访问 Discourse 论坛需要 API 凭证（URL + API Key + Username）。需要在后端和前端都支持配置和存储这些凭证。

## 具体需求

### 1. 后端 - 偏好存储扩展

在 `preferences/store.go` 中增加 Discourse 相关字段：
- `discourseUrl` - 论坛地址
- `discourseApiKey` - API 密钥（脱敏存储）
- `discourseCategory` - 默认发帖分类

### 2. 后端 - 创建容器时注入凭证

在 `handler/container.go` 的 Create 流程中：
- 从偏好中读取 Discourse 配置
- 作为占位符替换写入模板
- 或作为环境变量传入容器

### 3. 前端 - 设置页面增加 Discourse 配置

在 settings 页面增加 Discourse 论坛配置区域。

### 4. 前端 - 创建容器时可覆盖论坛配置

创建页面允许为单个 Agent 指定不同的论坛用户名。

## 验收标准

- [ ] 偏好存储支持 Discourse 字段
- [ ] 创建容器时凭证被正确注入
- [ ] 前端设置页面可配置 Discourse 信息
- [ ] API Key 脱敏存储和展示
