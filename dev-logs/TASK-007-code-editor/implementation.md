# TASK-007 集成开源代码编辑器 - 实现记录

> 完成时间: 2026-03-11

## 方案

使用 `@monaco-editor/react` 替换原有的 textarea 编辑器，获得 VS Code 级别的编辑体验。

## 变更文件

- `frontend/src/components/CodeEditor.tsx` — 用 Monaco Editor 重写
- `frontend/src/app/containers/[id]/files/page.tsx` — 传入 filePath 用于语言检测

## 功能

- 根据文件扩展名自动识别语言（js/ts/json/yaml/py/go/md/html/css 等 20+ 种）
- vs-dark 暗色主题，与项目 UI 风格一致
- 括号配对着色、行高亮、平滑滚动
- 保留 Ctrl+S 保存快捷键
- 自动布局适配容器大小
