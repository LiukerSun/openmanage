"use client";

import { useState, useRef } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface AgentProgress {
  agent: string;
  index: number;
  step: string;
  status: "running" | "done" | "error" | "warn";
  message: string;
}

export default function BatchCreatePage() {
  const router = useRouter();
  const [prefix, setPrefix] = useState("agent");
  const [count, setCount] = useState("5");
  const [startPort, setStartPort] = useState("18790");
  const [description, setDescription] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");
  const [agents, setAgents] = useState<Record<string, AgentProgress>>({});
  const [summary, setSummary] = useState<{ total: number; success: number; failed: number } | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const updateAgent = (name: string, data: AgentProgress) => {
    setAgents((prev) => ({ ...prev, [name]: data }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!prefix.trim()) { setError("请输入名称前缀"); return; }

    setCreating(true);
    setError("");
    setAgents({});
    setSummary(null);

    const abort = new AbortController();
    abortRef.current = abort;

    try {
      const res = await fetch(`${API_BASE}/api/batch-create`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          prefix: prefix.trim(),
          count: parseInt(count) || 5,
          startPort: parseInt(startPort) || 18790,
          description: description.trim() || undefined,
        }),
        signal: abort.signal,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        throw new Error(err.error || res.statusText);
      }

      const reader = res.body?.getReader();
      if (!reader) throw new Error("无法读取响应流");

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
              setSummary({ total: evt.total, success: evt.success, failed: evt.failed });
            } else if (evt.agent) {
              updateAgent(evt.agent, evt);
            }
          } catch {}
        }
      }
    } catch (e: any) {
      if (e.name !== "AbortError") setError(e.message);
    }
    setCreating(false);
  };

  const stepIcon = (status: string) => {
    switch (status) {
      case "running": return <span className="text-blue-400 animate-pulse">◉</span>;
      case "done": return <span className="text-green-400">✓</span>;
      case "warn": return <span className="text-yellow-400">⚠</span>;
      case "error": return <span className="text-red-400">✗</span>;
      default: return <span className="text-gray-600">○</span>;
    }
  };

  const agentList = Object.values(agents).sort((a, b) => a.index - b.index);

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-bold mb-6">批量创建 Agent</h1>
      {error && <div className="mb-4 p-3 bg-red-900/50 border border-red-700 rounded text-red-300 text-sm">{error}</div>}

      {summary && (
        <div className="mb-6 p-4 bg-gray-900/80 border border-green-800 rounded">
          <div className="text-green-400 font-medium mb-1">批量创建完成</div>
          <div className="text-sm text-gray-400">
            共 {summary.total} 个 · 成功 <span className="text-green-400">{summary.success}</span> 个
            {summary.failed > 0 && <> · 失败 <span className="text-red-400">{summary.failed}</span> 个</>}
          </div>
          <Link href="/" className="inline-block mt-3 px-4 py-2 bg-blue-700 hover:bg-blue-600 rounded text-sm">返回首页</Link>
        </div>
      )}

      {agentList.length > 0 && !summary && (
        <div className="mb-6 p-4 bg-gray-900/80 border border-gray-700 rounded space-y-2 max-h-80 overflow-y-auto">
          <div className="text-sm text-gray-400 mb-2">创建进度 ({agentList.filter(a => a.step === "done").length}/{count})</div>
          {agentList.map((a) => (
            <div key={a.agent} className="flex items-center gap-2 text-sm">
              <span className="w-4 text-center flex-shrink-0">{stepIcon(a.status)}</span>
              <span className="text-gray-300 font-mono w-28 flex-shrink-0">{a.agent}</span>
              <span className={`text-xs truncate ${a.status === "error" ? "text-red-400" : "text-gray-500"}`}>{a.message}</span>
            </div>
          ))}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-3 gap-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">名称前缀 *</label>
            <input value={prefix} onChange={(e) => setPrefix(e.target.value)} placeholder="agent" disabled={creating} className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-600 disabled:opacity-50" />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">数量 (1-20)</label>
            <input value={count} onChange={(e) => setCount(e.target.value)} type="number" min="1" max="20" disabled={creating} className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-600 disabled:opacity-50" />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">起始端口</label>
            <input value={startPort} onChange={(e) => setStartPort(e.target.value)} type="number" disabled={creating} className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-600 disabled:opacity-50" />
          </div>
        </div>
        <div>
          <label className="block text-sm text-gray-400 mb-1">Agent 描述</label>
          <textarea value={description} onChange={(e) => setDescription(e.target.value)} placeholder="统一描述所有 Agent 的用途、性格等，AI 将为每个 Agent 生成个性化配置。留空则使用默认模板。" rows={3} disabled={creating} className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-600 resize-none disabled:opacity-50" />
        </div>
        <p className="text-xs text-gray-600">将创建 {prefix}-1 ~ {prefix}-{count}，端口 {startPort} ~ {parseInt(startPort) + parseInt(count) - 1}</p>
        <div className="flex gap-3 pt-2">
          <button type="submit" disabled={creating} className="px-4 py-2 bg-blue-700 hover:bg-blue-600 disabled:opacity-50 rounded text-sm">
            {creating ? "创建中..." : `批量创建 ${count} 个 Agent`}
          </button>
          <Link href="/" className={`px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded text-sm ${creating ? "pointer-events-none opacity-50" : ""}`}>取消</Link>
        </div>
      </form>
    </div>
  );
}
