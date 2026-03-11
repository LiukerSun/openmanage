# TASK-008: OpenClaw 配置模板自动生成

## 需求背景

当前创建容器时，`copyTemplates` 只做简单的 `{{NAME}}` 替换，且现有的 `openclaw.json` 模板内容过于简单。用户提供了一份完整的 OpenClaw 配置模板，需要在创建容器时自动生成完整配置，其中部分字段需要动态生成。

## 需求描述

将用户提供的完整 OpenClaw 配置作为模板，创建容器时自动替换动态字段：

### 需要自动生成的字段（5 个）

| 字段 | 生成方式 |
|------|---------|
| `wizard.lastRunAt` | 容器创建时的 ISO 8601 时间戳 |
| `agents.defaults.workspace` | 容器内工作目录，基于容器数据路径拼接 |
| `gateway.port` | 使用用户指定的端口（来自 CreateContainerRequest.Port） |
| `gateway.auth.token` | 随机生成 48 字符 hex 字符串 |
| `meta.lastTouchedAt` | 容器创建时的 ISO 8601 时间戳 |

### 保持不变的字段

其余所有字段保持模板原值不变，包括：
- `auth` 认证配置
- `models` 模型提供商和模型列表
- `agents.defaults.model` 默认模型
- `tools` 工具配置（含 Brave Search API Key）
- `gateway.auth.mode`、`gateway.bind` 等固定配置
- `commands`、`messages`、`session`、`channels` 等

## 验收标准

1. 创建容器时，数据目录中生成完整的 OpenClaw 配置文件
2. 动态字段正确替换（时间戳、token、端口、workspace 路径）
3. 静态字段保持模板原值
4. 已存在配置文件时不覆盖（保持现有 skip 逻辑）
