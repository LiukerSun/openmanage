# TASK-021: 模型接入商配置

## 需求
在设置页新增模型接入商配置，支持预定义列表选择，配置 API Key 后替换当前硬编码的 GLM API。

## 预定义接入商列表

| ID | 名称 | Base URL | 默认模型 |
|----|------|----------|---------|
| zhipu | 智谱 GLM | https://open.bigmodel.cn/api/coding/paas/v4 | glm-4.7-flash |
| openai | OpenAI | https://api.openai.com/v1 | gpt-4o-mini |
| anthropic | Anthropic | https://api.anthropic.com/v1 | claude-sonnet-4-20250514 |
| deepseek | DeepSeek | https://api.deepseek.com/v1 | deepseek-chat |
| qwen | 通义千问 | https://dashscope.aliyuncs.com/compatible-mode/v1 | qwen-plus |

## 改动范围

### 1. 后端 preferences/store.go
- UserPreferences 新增 `ModelProviders []ModelProvider` 字段
- ModelProvider: `{id, name, baseUrl, apiKey, model, enabled}`
- Save 时对 apiKey 做 mask 保留逻辑

### 2. 后端 ai/client.go
- Client 构造函数改为接收 baseURL + apiKey + model
- 去掉硬编码的 const baseURL/model

### 3. 后端 main.go
- 启动时从 preferences 读取 enabled 的 provider 初始化 AI client
- 保留 GLM_API_KEY 环境变量作为 fallback

### 4. 前端 api.ts
- UserPreferences 接口新增 modelProviders 字段

### 5. 前端 settings/page.tsx
- 新增"模型接入商"配置卡片
- 预定义列表 checkbox 选择 + API Key 输入
