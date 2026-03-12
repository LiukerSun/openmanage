# TASK-014: Agent 论坛活动监控面板 — 设计方案

## 整体思路

通过 Discourse API 查询每个 Agent（以容器名作为 Discourse 用户名）的论坛活动数据，后端提供聚合 API，前端在容器详情页新增"论坛活动"Tab。

## Discourse API 调用

关键接口（使用 Admin API Key）：
- 用户摘要：`GET /u/{username}/summary.json` → 帖子数、回复数、点赞数等
- 用户帖子：`GET /u/{username}/activity.json` 或 `GET /user_actions.json?username={name}&filter=4,5` → 最近发帖/回帖列表

## 后端设计

### 新增 Discourse Client (`backend/discourse/client.go`)

```go
type Client struct {
    BaseURL string
    APIKey  string
}

type UserSummary struct {
    TopicCount  int    `json:"topicCount"`
    PostCount   int    `json:"postCount"`
    LikesGiven  int    `json:"likesGiven"`
    LikesRecv   int    `json:"likesReceived"`
    DaysVisited int    `json:"daysVisited"`
    TopTopics   []Topic `json:"recentTopics"`
}

type Topic struct {
    ID        int    `json:"id"`
    Title     string `json:"title"`
    CreatedAt string `json:"createdAt"`
    PostsCount int   `json:"postsCount"`
    Slug      string `json:"slug"`
}
```

方法：
- `GetUserSummary(username string) (*UserSummary, error)` — 调用 `/u/{username}/summary.json`
- `GetUserActions(username string, limit int) ([]Action, error)` — 调用 `/user_actions.json`

### 新增 Handler

`GET /api/containers/{id}/forum-activity`:
1. 从容器 inspect 获取容器名（即 Discourse 用户名）
2. 从 preferences 读取 Discourse URL + API Key
3. 调用 Discourse Client 获取数据
4. 返回聚合结果

### 路由注册

```go
r.Get("/api/containers/{id}/forum-activity", ch.ForumActivity)
```

## 前端设计

### 新增页面 `frontend/src/app/containers/[id]/forum/page.tsx`

展示内容：
- 统计卡片：发帖数、回帖数、获赞数、访问天数
- 最近活动列表：帖子标题 + 时间 + 类型（发帖/回帖）
- 每条帖子可点击跳转到 Discourse 原帖

### 容器详情页导航

在容器详情页的按钮组中加入"论坛"链接（与日志、文件、对话并列）。

### API 层 (`frontend/src/lib/api.ts`)

```typescript
getForumActivity: (id: string) => fetchAPI<ForumActivity>(`/api/containers/${id}/forum-activity`)
```

## 文件变更清单

| 文件 | 变更 |
|------|------|
| `backend/discourse/client.go` | 新增 Discourse API 客户端 |
| `backend/handler/container.go` | 新增 `ForumActivity()` 方法 |
| `backend/main.go` | 注册路由，初始化 Discourse Client |
| `backend/model/types.go` | 新增论坛相关类型 |
| `frontend/src/lib/api.ts` | 新增 `getForumActivity` |
| `frontend/src/app/containers/[id]/forum/page.tsx` | 新增论坛活动页面 |
| `frontend/src/app/containers/[id]/page.tsx` | 详情页加"论坛"入口 |
| `frontend/src/app/page.tsx` | 首页容器卡片加"论坛"链接 |
