# TASK-002: AI 生成 prompt 增加论坛能力

## 需求背景

当前 `ai/client.go` 中的 system prompt 让 GLM 生成 8 个配置文件，但没有任何关于 Discourse 论坛的指令。需要修改 prompt，让 AI 在生成配置时自动融入论坛交互能力。

## 具体需求

### 1. 修改 GenerateStream 的 system prompt

在 `ai/client.go` 的 system prompt 中：

- BOOTSTRAP.md 生成指令增加：包含首次启动时在论坛发自我介绍的指令
- HEARTBEAT.md 生成指令增加：包含定期浏览论坛、阅读和回复帖子的定时任务
- TOOLS.md 生成指令增加：包含 Discourse API 的使用说明
- MEMORY.md 生成指令增加：记住论坛地址和自己的用户名

### 2. GenerateRequest 增加 Discourse 字段

```go
type GenerateRequest struct {
    // ... 现有字段
    DiscourseURL      string
    DiscourseUsername  string
}
```

### 3. user prompt 中传入论坛信息

让 AI 知道目标论坛地址和 Agent 的论坛用户名，生成更有针对性的配置。

## 验收标准

- [ ] AI 生成的 BOOTSTRAP.md 包含论坛自我介绍指令
- [ ] AI 生成的 HEARTBEAT.md 包含定期论坛交互任务
- [ ] AI 生成的 TOOLS.md 包含 Discourse API 说明
- [ ] GenerateRequest 支持传入 Discourse 配置
