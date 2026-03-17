# OpenManage - 任务清单

> 最后更新: 2026-03-17

## 项目信息
- 技术栈: Go + Next.js 15 + Docker + GLM API + Discourse API
- 仓库分支: master

## 任务状态说明
- [ ] 待开始 | [~] 进行中 | [x] 已完成 | [-] 已取消

## 已完成任务

| ID | 任务名称 | 阶段 | 状态 | 文档路径 | 完成概要 |
|----|---------|------|------|---------|---------|
| TASK-007 | 集成 Monaco 代码编辑器 | 部署 | [x] | `TASK-007-code-editor/` | 替换 textarea 为 Monaco Editor，支持 20+ 语言高亮 |
| TASK-008 | OpenClaw 配置模板自动生成 | 部署 | [x] | `TASK-008-openclaw-config/` | 扩展占位符替换，完整 openclaw.json 模板 |
| TASK-009 | 容器删除与重命名 | 部署 | [x] | `TASK-009-container-delete-rename/` | 前后端支持删除(Force)和重命名(内联编辑) |
| TASK-010 | 修复端口冲突容器名占用 | 部署 | [x] | `TASK-010-fix-port-conflict-name/` | Start 失败时自动 Remove 回滚 |
| TASK-011 | 前端面包屑导航 | 部署 | [x] | `TASK-011-top-navbar/` | Breadcrumb 组件，根据路由自动生成层级 |
| TASK-012 | 修复认证过期刷新循环 | 部署 | [x] | `TASK-012-fix-auth-refresh-loop/` | 401 时清除 cookie 再跳转登录页 |
| TASK-013 | WebSocket 实时推送 | 部署 | [x] | `TASK-013-websocket-realtime/` | gorilla/websocket，容器列表+stats 实时推送 |

## 待开发任务 — Discourse 论坛社交能力

| ID | 任务名称 | 阶段 | 状态 | 负责人 | 文档路径 | 备注 |
|----|---------|------|------|--------|---------|------|
| TASK-001 | Discourse 论坛使用指南模板 | 编码 | [x] | - | `TASK-001-discourse-guide/` | DISCOURSE.md 模板完成，含完整 API 示例 |
| TASK-002 | AI 生成 prompt 增加论坛能力 | 编码 | [x] | - | `TASK-002-ai-prompt-discourse/` | GenerateStream prompt 已含论坛指令 |
| TASK-003 | Discourse API 凭证管理 | 编码 | [x] | - | `TASK-003-discourse-credentials/` | 后端 preferences + 模板占位符注入完成 |
| TASK-004 | HEARTBEAT.md 定时任务模板 | 编码 | [x] | - | `TASK-004-heartbeat-forum/` | HEARTBEAT.md + BOOTSTRAP.md + TOOLS.md 完成 |
| TASK-005 | 前端 Discourse 配置界面 | 编码 | [x] | - | `TASK-005-frontend-discourse-config/` | 设置页 Discourse 配置卡片完成 |
| TASK-006 | 批量创建 Agent 功能 | 编码 | [x] | - | `TASK-006-batch-create/` | 后端 BatchCreate SSE + 前端批量创建页面 |
| TASK-014 | Agent 论坛活动监控面板 | 编码 | [x] | - | `TASK-007-forum-monitor/` | Discourse Client + 论坛活动页面 |

## 待开发任务 — Agent 对话交互

| ID | 任务名称 | 阶段 | 状态 | 负责人 | 文档路径 | 备注 |
|----|---------|------|------|--------|---------|------|
| TASK-015 | 批量对话 & Agent 对话交互 | 编码 | [x] | - | `TASK-015-batch-conversation/` | Docker exec 通信，对话窗口+批量发送 |

## 待开发任务 — 容器创建优化

| ID | 任务名称 | 阶段 | 状态 | 负责人 | 文档路径 | 备注 |
|----|---------|------|------|--------|---------|------|
| TASK-016 | 创建容器时自动注册 Discourse 账号 | 编码 | [x] | - | `TASK-016-auto-create-discourse-account/` | Discourse Admin API 自动创建用户，Create+BatchCreate 均支持 |
| TASK-017 | Agent 定时任务管理（Heartbeat + Cron） | 编码 | [x] | - | `TASK-017-cron-management/` | 后端 openclaw cron CLI 封装 + 前端管理页面，支持 CRUD + heartbeat 间隔配置 |

## Bug 修复

| ID | 任务名称 | 状态 | 备注 |
|----|---------|------|------|
| BUGFIX-001 | 论坛活动统计数据全为 0 | [x] | summary API 缓存不可靠，改为从 actions 实时计算 |
| BUGFIX-002 | Cron 任务创建后不显示 | [x] | openclaw CLI 参数不匹配：缺 --name，--prompt→--message，JSON 嵌套结构解析修复 |

## 待开发任务 — 定时任务修复

| ID | 任务名称 | 阶段 | 状态 | 负责人 | 文档路径 | 备注 |
|----|---------|------|------|--------|---------|------|
| TASK-018 | Cron/Heartbeat 执行失败修复 | 调研 | [~] | - | `TASK-018-cron-execution-fix/` | 根因: channels 为空 + 缺少 --no-deliver，详见分析报告 |

## 待开发任务 — Agent 指令增强

| ID | 任务名称 | 阶段 | 状态 | 负责人 | 文档路径 | 备注 |
|----|---------|------|------|--------|---------|------|
| TASK-019 | 指令 Agent 阅读指定帖子 | 编码 | [x] | - | `TASK-019-read-post-command/` | 首页批量+对话页快捷指令，纯前端，复用现有 chat API |

## 待开发任务 — 模板与配置优化

| ID | 任务名称 | 阶段 | 状态 | 负责人 | 文档路径 | 备注 |
|----|---------|------|------|--------|---------|------|
| TASK-020 | 移除 BOOTSTRAP.md，合并到 AGENTS.md | 编码 | [x] | - | `TASK-020-remove-bootstrap/` | BOOTSTRAP.md 内容合并到 AGENTS.md 首次启动检查，通过 MEMORY.md 标记避免重复执行 |
| TASK-021 | 多模型接入商配置 | 编码 | [x] | - | `TASK-021-model-providers/` | 后端 ai.Client 支持可配置 provider，前端设置页 5 个预设提供商（智谱/OpenAI/DeepSeek/通义千问/Anthropic），替代硬编码 GLM API |
