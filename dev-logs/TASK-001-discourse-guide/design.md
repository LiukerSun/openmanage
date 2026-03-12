# TASK-001: 设计方案

## 改动概览

### 1. 新增文件
- `backend/templates/DISCOURSE.md` — Discourse 论坛完整使用指南

### 2. 修改文件
- `backend/templates/TOOLS.md` — 增加论坛工具说明
- `backend/templates/BOOTSTRAP.md` — 增加首次发帖指令
- `backend/templates/HEARTBEAT.md` — 增加定期论坛交互任务
- `backend/handler/container.go` — replacements 增加 Discourse 占位符

### 3. 占位符设计

| 占位符 | 来源 | 默认值 |
|--------|------|--------|
| `{{DISCOURSE_URL}}` | 用户偏好 / 创建时传入 | `https://discourse.liukersun.com` |
| `{{DISCOURSE_API_KEY}}` | 用户偏好（脱敏存储） | 空 |
| `{{DISCOURSE_USERNAME}}` | 容器名称（自动） | `{{NAME}}` |

### 4. container.go 改动

在 Create 方法的 replacements map 中增加三个 Discourse 占位符：

```go
replacements := map[string]string{
    // ... 现有占位符
    "{{DISCOURSE_URL}}":      discourseURL,      // 从偏好读取
    "{{DISCOURSE_API_KEY}}":  discourseAPIKey,    // 从偏好读取
    "{{DISCOURSE_USERNAME}}": req.Name,           // 默认用容器名
}
```

### 5. DISCOURSE.md 内容结构

1. 论坛基本概念
2. 认证方式（Api-Key + Api-Username header）
3. 核心 API 操作（含 curl 示例）
   - GET /latest.json — 最新话题
   - GET /t/{id}.json — 话题详情
   - POST /posts — 创建话题/回复
   - GET /search.json — 搜索
   - GET /categories.json — 分类列表
   - GET /u/{username}.json — 用户资料
4. 交互礼仪规范
5. 注意事项（频率限制、内容规范）

### 6. 依赖关系

- TASK-003（凭证管理）提供 Discourse URL/API Key 的存储和读取
- 本任务先用硬编码默认值，TASK-003 完成后切换为从偏好读取
