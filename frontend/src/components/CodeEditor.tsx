"use client";

import { useRef, useCallback } from "react";
import Editor, { OnMount } from "@monaco-editor/react";
import type { editor } from "monaco-editor";

interface CodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  onKeyDown?: (e: React.KeyboardEvent) => void;
  filePath?: string;
}

/** 根据文件扩展名推断 Monaco 语言 */
function getLanguage(filePath?: string): string {
  if (!filePath) return "plaintext";
  const ext = filePath.split(".").pop()?.toLowerCase();
  const map: Record<string, string> = {
    js: "javascript",
    jsx: "javascript",
    ts: "typescript",
    tsx: "typescript",
    json: "json",
    md: "markdown",
    yaml: "yaml",
    yml: "yaml",
    xml: "xml",
    html: "html",
    css: "css",
    scss: "scss",
    less: "less",
    py: "python",
    go: "go",
    rs: "rust",
    java: "java",
    sh: "shell",
    bash: "shell",
    zsh: "shell",
    sql: "sql",
    dockerfile: "dockerfile",
    toml: "ini",
    ini: "ini",
    env: "ini",
    graphql: "graphql",
    vue: "html",
    svelte: "html",
  };
  return map[ext ?? ""] ?? "plaintext";
}

export function CodeEditor({ value, onChange, onKeyDown, filePath }: CodeEditorProps) {
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const handleMount: OnMount = useCallback((ed) => {
    editorRef.current = ed;
    // Ctrl+S / Cmd+S 保存
    ed.addCommand(
      // Monaco KeyMod.CtrlCmd | KeyCode.KeyS
      2048 | 49, // CtrlCmd = 2048, KeyS = 49
      () => {
        if (!onKeyDown) return;
        const syntheticEvent = new KeyboardEvent("keydown", {
          key: "s",
          ctrlKey: true,
          bubbles: true,
        });
        // 触发外部 onKeyDown
        wrapperRef.current?.dispatchEvent(syntheticEvent);
      }
    );
  }, [onKeyDown]);

  const handleChange = useCallback((val: string | undefined) => {
    onChange(val ?? "");
  }, [onChange]);

  return (
    <div
      ref={wrapperRef}
      className="flex-1 rounded-lg border border-gray-800 overflow-hidden focus-within:border-blue-600 transition-colors"
      onKeyDown={onKeyDown}
    >
      <Editor
        height="100%"
        language={getLanguage(filePath)}
        value={value}
        onChange={handleChange}
        onMount={handleMount}
        theme="vs-dark"
        options={{
          fontSize: 13,
          lineHeight: 20,
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          wordWrap: "on",
          tabSize: 2,
          automaticLayout: true,
          padding: { top: 12, bottom: 12 },
          renderLineHighlight: "line",
          cursorBlinking: "smooth",
          smoothScrolling: true,
          bracketPairColorization: { enabled: true },
        }}
        loading={
          <div className="flex items-center justify-center h-full text-gray-500 text-sm">
            编辑器加载中...
          </div>
        }
      />
    </div>
  );
}
