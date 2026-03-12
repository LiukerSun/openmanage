# TASK-016: 创建容器时自动注册 Discourse 账号

## 需求背景
当前 Agent 启动后需要自己通过 API 注册 Discourse 账号，但经常遇到困难（注册流程复杂、需要邮箱验证等）。希望在创建容器时由 OpenManage 后端通过全局 Admin API Key 预先创建好账号，Agent 启动后直接可用。

## 核心需求
在 `Create` 和 `BatchCreate` 流程中，模板复制之后、容器创建之前，新增一个步骤：
1. 调用 Discourse Admin API (`POST /users.json`) 创建用户
2. 参数: username=agentName, email=agentName@agents.local, password=随机生成, active=true, approved=true
3. 如果用户已存在（409 或返回错误），跳过不报错
4. 创建成功后通过 SSE 通知前端

## Discourse API 细节
- Endpoint: `POST {discourseURL}/users.json`
- Headers: `Api-Key`, `Api-Username: system`
- Body (form-encoded): name, email, password, username, active=true, approved=true
- 需要全局 Admin API Key（已有，存在 preferences 中）

## 改动范围
1. `backend/discourse/client.go` — 新增 `CreateUser(username, email, password string) error`
2. `backend/handler/container.go` — Create() 和 BatchCreate() 中插入注册步骤
3. TOOLS.md 模板 — 告知 Agent 账号已预创建，无需自行注册
