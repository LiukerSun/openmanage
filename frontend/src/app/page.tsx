"use client";

import { useState } from "react";
import { api } from "@/lib/api";
import { useContainerList } from "@/lib/ws";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import Link from "next/link";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

const actionLabels = {
  start: "启动",
  stop: "停止",
  restart: "重启",
  delete: "删除",
} as const;

const actionVariants = {
  start: "default",
  stop: "danger",
  restart: "warning",
  delete: "danger",
} as const;

interface BatchProgress {
  agent: string;
  index: number;
  step: string;
  status: "running" | "done" | "error";
  message: string;
}

export default function Dashboard() {
  const { containers, loading } = useContainerList();
  const [error, setError] = useState("");
  const [confirm, setConfirm] = useState<{ id: string; name: string; act: "start" | "stop" | "restart" | "delete" } | null>(null);
  const [acting, setActing] = useState(false);

  // Batch chat state
  const [selectMode, setSelectMode] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [batchInput, setBatchInput] = useState("");
  const [batchSending, setBatchSending] = useState(false);
  const [batchProgress, setBatchProgress] = useState<Record<string, BatchProgress>>({});
  const [batchSummary, setBatchSummary] = useState("");

  const requestAction = (id: string, name: string, act: "start" | "stop" | "restart" | "delete") => {
    if (act === "start") {
      doAction(id, act);
    } else {
      setConfirm({ id, name, act });
    }
  };

  const doAction = async (id: string, act: "start" | "stop" | "restart" | "delete") => {
    setActing(true);
    try {
      if (act === "start") await api.startContainer(id);
      else if (act === "stop") await api.stopContainer(id);
      else if (act === "restart") await api.restartContainer(id);
      else if (act === "delete") await api.deleteContainer(id);
    } catch (e: any) { setError(e.message); }
    setActing(false);
    setConfirm(null);
  };

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const selectAllRunning = () => {
    const running = containers.filter((c) => c.state === "running").map((c) => c.id);
    setSelected(new Set(running));
  };

  const exitSelectMode = () => {
    setSelectMode(false);
    setSelected(new Set());
    setBatchInput("");
    setBatchProgress({});
    setBatchSummary("");
  };

  const sendBatchChat = async () => {
    if (selected.size === 0 || !batchInput.trim() || batchSending) return;
    setBatchSending(true);
    setBatchProgress({});
    setBatchSummary("");
    setError("");

    try {
      const res = await fetch(`${API_BASE}/api/batch/chat`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ containerIds: Array.from(selected), message: batchInput.trim() }),
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        throw new Error(err.error || res.statusText);
      }

      const reader = res.body?.getReader();
      if (!reader) throw new Error("No response stream");

      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });

        const lines = buffer.split("\n");
        buffer = lines.pop() || "";

        for (const line of lines) {
          if (!line.startsWith("data: ")) continue;
          try {
            const evt = JSON.parse(line.slice(6));
            if (evt.step === "batch-done") {
              setBatchSummary(evt.message);
            } else if (evt.agent) {
              setBatchProgress((prev) => ({ ...prev, [evt.agent]: evt }));
            }
          } catch {
            // ignore
          }
        }
      }
    } catch (e: unknown) {
      const errMsg = e instanceof Error ? e.message : "批量发送失败";
      setError(errMsg);
    } finally {
      setBatchSending(false);
    }
  };

  // Map container ID (first 12 chars) to name for progress display
  const idToName: Record<string, string> = {};
  containers.forEach((c) => { idToName[c.id.slice(0, 12)] = c.name || c.id.slice(0, 12); });

  if (loading) return <div className="text-gray-400">加载中...</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">OpenClaw 容器</h1>
        <div className="flex gap-2">
          {!selectMode ? (
            <button onClick={() => setSelectMode(true)} className="px-4 py-2 bg-teal-700 hover:bg-teal-600 rounded text-sm">
              批量指令
            </button>
          ) : (
            <button onClick={exitSelectMode} className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded text-sm">
              退出选择
            </button>
          )}
          <Link href="/create" className="px-4 py-2 bg-blue-700 hover:bg-blue-600 rounded text-sm">+ 新建容器</Link>
          <Link href="/batch-create" className="px-4 py-2 bg-purple-700 hover:bg-purple-600 rounded text-sm">批量创建</Link>
        </div>
      </div>

      {error && <div className="mb-3 text-sm text-red-400">{error}</div>}

      {containers.length === 0 ? (
        <p className="text-gray-500">未找到 OpenClaw 容器。</p>
      ) : (
        <div className="grid gap-4">
          {containers.map((c) => {
            const isRunning = c.state === "running";
            const isSelected = selected.has(c.id);
            return (
              <div
                key={c.id}
                className={`bg-gray-900 border rounded-lg p-4 ${selectMode && isSelected ? "border-teal-500" : "border-gray-800"}`}
                onClick={selectMode && isRunning ? () => toggleSelect(c.id) : undefined}
                style={selectMode ? { cursor: isRunning ? "pointer" : "not-allowed" } : undefined}
              >
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-3">
                    {selectMode && (
                      <input
                        type="checkbox"
                        checked={isSelected}
                        disabled={!isRunning}
                        onChange={() => toggleSelect(c.id)}
                        onClick={(e) => e.stopPropagation()}
                        className="w-4 h-4 accent-teal-500"
                      />
                    )}
                    <div>
                      <Link href={`/containers/${c.id}`} className="text-lg font-semibold hover:text-blue-400" onClick={(e) => selectMode && e.preventDefault()}>
                        {c.name || c.id}
                      </Link>
                      <p className="text-sm text-gray-500">{c.image}</p>
                    </div>
                  </div>
                  <span className={`px-2 py-1 rounded text-xs font-medium ${isRunning ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300"}`}>
                    {isRunning ? "运行中" : c.state === "exited" ? "已停止" : c.state}
                  </span>
                </div>
                {!selectMode && (
                  <div className="flex flex-wrap gap-2">
                    <button onClick={() => requestAction(c.id, c.name || c.id, "start")} disabled={isRunning} className="px-3 py-1 bg-green-700 hover:bg-green-600 disabled:opacity-30 rounded text-sm">启动</button>
                    <button onClick={() => requestAction(c.id, c.name || c.id, "stop")} disabled={!isRunning} className="px-3 py-1 bg-red-700 hover:bg-red-600 disabled:opacity-30 rounded text-sm">停止</button>
                    <button onClick={() => requestAction(c.id, c.name || c.id, "restart")} disabled={!isRunning} className="px-3 py-1 bg-yellow-700 hover:bg-yellow-600 disabled:opacity-30 rounded text-sm">重启</button>
                    <button onClick={() => requestAction(c.id, c.name || c.id, "delete")} className="px-3 py-1 bg-red-900 hover:bg-red-800 rounded text-sm">删除</button>
                    <Link href={`/containers/${c.id}/logs`} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">日志</Link>
                    <Link href={`/containers/${c.id}/files`} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">文件</Link>
                    <Link href={`/containers/${c.id}/conversations`} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">对话</Link>
                    <Link href={`/containers/${c.id}/forum`} className="px-3 py-1 bg-indigo-700 hover:bg-indigo-600 rounded text-sm">论坛</Link>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Batch chat bar */}
      {selectMode && (
        <div className="fixed bottom-0 left-0 right-0 bg-gray-900 border-t border-gray-700 p-4 z-50">
          <div className="max-w-5xl mx-auto">
            <div className="flex items-center gap-3 mb-3">
              <span className="text-sm text-gray-400">已选 {selected.size} 个 Agent</span>
              <button onClick={selectAllRunning} className="text-sm text-teal-400 hover:text-teal-300">全选运行中</button>
              <button onClick={() => setSelected(new Set())} className="text-sm text-gray-500 hover:text-gray-400">清空</button>
            </div>
            <div className="flex gap-2">
              <input
                value={batchInput}
                onChange={(e) => setBatchInput(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && sendBatchChat()}
                placeholder="输入要发送给所有选中 Agent 的指令..."
                className="flex-1 bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-teal-600"
                disabled={batchSending}
              />
              <button
                onClick={sendBatchChat}
                disabled={batchSending || selected.size === 0 || !batchInput.trim()}
                className="px-6 py-2 bg-teal-700 hover:bg-teal-600 disabled:bg-gray-700 disabled:text-gray-500 rounded text-sm whitespace-nowrap"
              >
                {batchSending ? "发送中..." : "发送指令"}
              </button>
            </div>

            {/* Progress */}
            {Object.keys(batchProgress).length > 0 && (
              <div className="mt-3 space-y-1 max-h-40 overflow-y-auto">
                {Object.values(batchProgress)
                  .sort((a, b) => a.index - b.index)
                  .map((p) => (
                    <div key={p.agent} className="flex items-center gap-2 text-sm">
                      <span className={`w-2 h-2 rounded-full ${p.status === "running" ? "bg-yellow-400 animate-pulse" : p.status === "done" ? "bg-green-400" : "bg-red-400"}`} />
                      <span className="text-gray-300 w-32 truncate">{idToName[p.agent] || p.agent}</span>
                      <span className="text-gray-500">{p.message}</span>
                    </div>
                  ))}
              </div>
            )}
            {batchSummary && (
              <div className="mt-2 text-sm text-teal-400">{batchSummary}</div>
            )}
          </div>
        </div>
      )}

      <ConfirmDialog
        open={!!confirm}
        title={`${confirm ? actionLabels[confirm.act] : ""}容器`}
        message={confirm ? <>确定要{actionLabels[confirm.act]}容器 <span className="text-white font-medium">{confirm.name}</span> 吗？</> : ""}
        confirmLabel={confirm ? actionLabels[confirm.act] : "确认"}
        variant={confirm ? actionVariants[confirm.act] : "default"}
        loading={acting}
        onConfirm={() => confirm && doAction(confirm.id, confirm.act)}
        onCancel={() => setConfirm(null)}
      />

      {/* Bottom padding when batch bar is visible */}
      {selectMode && <div className="h-48" />}
    </div>
  );
}
