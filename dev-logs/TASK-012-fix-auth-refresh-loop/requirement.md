# TASK-012: 修复认证过期后页面反复刷新

## 问题描述
JWT 过期后，页面不会跳转到登录界面，而是陷入无限刷新循环。

## 根因分析
1. JWT 过期 → 后端 API 返回 401
2. `fetchAPI` 收到 401 → 执行 `window.location.href = "/login"`
3. Next.js middleware 检查 cookie 中的 `auth_token`，过期 token 仍存在 → middleware 认为已登录，重定向回首页
4. 页面加载 → `AuthProvider.checkAuthStatus()` 再次触发 API 调用 → 又 401 → 循环

核心问题：401 时未清除 cookie 中的过期 token。

## 修复方案
在所有认证失败路径中，跳转登录页前先清除 `auth_token` cookie：
- `fetchAPI` 的 401 拦截
- `AuthProvider.checkAuthStatus` 失败分支
- `logout` / `logoutSimple` 函数

## 修改文件
- `frontend/src/lib/api.ts` — fetchAPI 401 处理时清除 cookie
- `frontend/src/lib/auth.tsx` — checkAuthStatus、logout、logoutSimple 清除 cookie
