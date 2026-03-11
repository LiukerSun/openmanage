# TASK-008: 设计方案

## 方案：扩展模板占位符替换

### 核心思路

沿用现有 `copyTemplates` 的字符串替换机制，在模板中使用占位符，创建时统一替换。

### 新增占位符

| 占位符 | 替换值 |
|--------|--------|
| `{{NAME}}` | 容器名称（已有） |
| `{{TIMESTAMP}}` | ISO 8601 时间戳 `time.Now().UTC().Format(time.RFC3339Nano)` |
| `{{WORKSPACE}}` | 容器内 workspace 路径 `/home/evan/.openclaw-<name>/workspace` |
| `{{PORT}}` | 端口号，来自 `req.Port` |
| `{{GATEWAY_TOKEN}}` | `crypto/rand` 生成 24 字节 → hex 编码 = 48 字符 |

### 改动点

1. `backend/templates/openclaw.json` — 替换为完整配置模板，动态字段用占位符
2. `backend/handler/container.go` — `copyTemplates` 函数签名扩展，支持多占位符替换
3. 无需改 model、前端、API 接口

### copyTemplates 改动

```go
// 改为接收 replacements map
func copyTemplates(templateDir, dataPath string, replacements map[string]string) error

// 调用处构造 replacements
replacements := map[string]string{
    "{{NAME}}":          req.Name,
    "{{TIMESTAMP}}":     time.Now().UTC().Format(time.RFC3339Nano),
    "{{WORKSPACE}}":     req.DataPath + "/workspace",
    "{{PORT}}":          strconv.Itoa(req.Port),
    "{{GATEWAY_TOKEN}}": generateToken(),
}
```
