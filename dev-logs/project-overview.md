# OpenClaw - 多 Agent 管理平台

> 项目概览文档 | 最后更新: 2026-03-11

## 项目背景

OpenClaw 是一个多 Agent 管理平台，用于管理和监控运行在 Docker 容器中的 AI Agent 实例。通过 Web 界面提供容器生命周期管理、日志查看、文件管理、对话记录查看等功能。

## 技术架构

```
┌─────────────────┐     ┌─────────────────┐
│   Frontend      │────▶│   Backend       │
│   Next.js 15    │     │   Go + Chi      │
│   React 19      │     │   Port: 8080    │
│   Tailwind 4    │     │                 │
│   Port: 3000    │     │   Docker SDK    │
└─────────────────┘     └────────┬────────┘
                                 │
                        ┌────────▼────────┐
                        │   Docker Engine │
                        │   (Agent 容器)   │
                        └─────────────────┘
```

## 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| 前端框架 | Next.js (App Router + Turbopack) | 15.3 |
| UI 框架 | React + Tailwind CSS | React 19 / Tailwind 4 |
| 后端框架 | Go + Chi Router | Go 1.26 / Chi 5.2 |
| 容器管理 | Docker SDK for Go | 28.5 |
| 认证 | JWT (golang-jwt) | 5.3 |
| 部署 | Docker Compose | - |

## 项目目录结构

```
openmanage/
├── docker-compose.yml              # 服务编排
├── backend/                        # Go 后端
│   ├── main.go                     # 入口，路由注册
│   ├── auth/                       # 认证逻辑 (store.go)
│   ├── docker/                     # Docker 客户端封装 (client.go)
│   ├── handler/                    # HTTP 处理器
│   │   ├── auth.go                 # 登录/登出/密码
│   │   ├── container.go            # 容器 CRUD + 生命周期
│   │   ├── conversations.go        # 对话记录
│   │   ├── files.go                # 容器文件管理
│   │   ├── logs.go                 # 日志流
│   │   └── templates.go            # Agent 模板管理
│   ├── middleware/                  # JWT 认证中间件
│   ├── model/                      # 数据模型 (types.go)
│   └── templates/                  # Agent 模板文件 (openclaw.json 等)
├── frontend/                       # Next.js 前端
│   └── src/
│       ├── app/                    # 页面路由
│       │   ├── page.tsx            # 首页（容器列表）
│       │   ├── login/              # 登录页
│       │   ├── settings/           # 设置页
│       │   ├── templates/          # 模板管理页
│       │   ├── create/             # 创建容器页
│       │   └── containers/[id]/    # 容器详情
│       │       ├── page.tsx        # 容器概览
│       │       ├── logs/           # 日志查看
│       │       ├── files/          # 文件管理
│       │       └── conversations/  # 对话记录
│       ├── components/             # 共享组件
│       │   ├── AppLayout.tsx       # 全局布局
│       │   └── UserMenu.tsx        # 用户菜单
│       ├── lib/                    # 工具库
│       │   ├── api.ts              # API 客户端
│       │   └── auth.tsx            # 认证上下文
│       └── middleware.ts           # Next.js 中间件（路由守卫）
└── dev-logs/                       # 开发日志（本文件所在）
```

## 已实现功能

### 后端 API
- 认证系统：登录/登出/密码修改/JWT 鉴权
- 容器管理：列表/详情/创建/启动/停止/重启
- 日志流：SSE 实时日志推送
- 文件管理：容器内文件浏览/读取/写入
- 对话记录：Agent 对话历史查看
- 模板管理：Agent 模板 CRUD

### 前端页面
- 登录页、首页（容器列表）、容器详情页
- 日志查看、文件管理、对话记录页面
- 模板管理页、设置页
- 全局布局 + 用户菜单 + 路由守卫

## 开发规范

- 后端遵循 Go 标准项目布局，handler 层处理 HTTP，业务逻辑在各模块内
- 前端使用 Next.js App Router，页面组件放在 `app/` 目录
- API 路径统一以 `/api/` 开头
- 认证使用 JWT，通过 Authorization header 传递
- Docker 容器通过挂载 `/var/run/docker.sock` 访问宿主机 Docker
