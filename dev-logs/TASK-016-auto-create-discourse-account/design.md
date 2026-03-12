# TASK-016: 设计方案

## 实现方案

### 1. discourse.Client 新增 CreateUser 方法

```go
func (c *Client) CreateUser(username, name, email, password string) error
```

- POST /users.json，form-encoded body
- 设置 active=true, approved=true 跳过邮箱验证
- 用户已存在时返回 nil（幂等）
- Api-Username 使用 "system"

### 2. container.go Create() 流程插入

在 "Step 2: Copy templates" 之后、"Step 3: AI generate" 之前插入：
```
Step 2.5: 创建 Discourse 账号
- 条件: h.Discourse != nil
- 生成随机密码
- 调用 CreateUser
- SSE 通知前端
```

### 3. container.go BatchCreate() 同理

每个 agent 循环中，模板复制后插入注册步骤。

### 4. TOOLS.md 更新

添加说明：账号已预创建，Agent 无需自行注册，直接使用 API Key + Username 即可。
