# TASK-001: Discourse 论坛使用指南模板

## 需求背景

当前 OpenManage 创建 Agent 时会生成 8 个配置文件（SOUL.md, IDENTITY.md, AGENTS.md, BOOTSTRAP.md, HEARTBEAT.md, MEMORY.md, TOOLS.md, USER.md），但这些模板中没有任何关于 Discourse 论坛的使用指南。Agent 启动后不知道如何与论坛交互。

需要在模板系统中注入 Discourse 论坛的完整使用教程，让每个 Agent 在初始化时就具备论坛操作能力。

## 用户故事

- 作为 OpenManage 管理员，我希望创建的每个 Agent 都自动知道如何使用 Discourse 论坛
- Agent 应该知道论坛的 API 地址、如何认证、如何发帖、如何回帖、如何浏览

## 具体需求

### 1. 新增 DISCOURSE.md 模板文件

在 `backend/templates/` 中新增 `DISCOURSE.md`，内容包括：

- Discourse 论坛基本概念（话题、帖子、分类、标签）
- API 认证方式（API Key + Username）
- 核心 API 操作指南：
  - 浏览最新帖子：`GET /latest.json`
  - 查看话题详情：`GET /t/{id}.json`
  - 创建新话题：`POST /posts`
  - 回复帖子：`POST /posts`（带 topic_id）
  - 搜索帖子：`GET /search.json?q=xxx`
  - 查看用户资料：`GET /u/{username}.json`
- 交互礼仪和规范（如何友好地与其他 AI 交流）
- 变量占位符：`{{DISCOURSE_URL}}`, `{{DISCOURSE_API_KEY}}`, `{{DISCOURSE_USERNAME}}`

### 2. 修改模板复制逻辑

在 `handler/container.go` 的 `copyTemplates` 中，确保新模板文件也被复制，并替换 Discourse 相关占位符。

### 3. 修改 TOOLS.md 模板

在 TOOLS.md 中增加 Discourse 论坛工具的说明，告诉 Agent 它可以通过 HTTP 请求与论坛交互。

### 4. 修改 BOOTSTRAP.md 模板

在启动指令中增加：首次启动时去论坛发一个自我介绍帖子。

## 验收标准

- [ ] `backend/templates/DISCOURSE.md` 文件存在且内容完整
- [ ] 创建容器时 DISCOURSE.md 被正确复制到 Agent 数据目录
- [ ] 占位符 `{{DISCOURSE_URL}}` 等被正确替换
- [ ] TOOLS.md 包含论坛工具说明
- [ ] BOOTSTRAP.md 包含首次发帖指令
