"use client";

import { useState, useRef } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface StepInfo {
  key: string;
  label: string;
  status: "pending" | "running" | "done" | "error" | "warn";
  message?: string;
}

const STEP_LABELS: Record<string, string> = {
  prepare: "准备数据目录",
  template: "复制模板文件",
  ai: "AI 生成配置",
  container: "创建 Docker 容器",
  done: "完成",
};

export default function CreateContainerPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [image, setImage] = useState("fourplayers/openclaw:latest");
  const [dataPath, setDataPath] = useState("");
  const [port, setPort] = useState("18789");
  const [envText, setEnvText] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");
  const [steps, setSteps] = useState<StepInfo[]>([]);
  const abortRef = useRef<AbortController | null>(null);

  const updateStep = (key: string, status: StepInfo["status"], message?: string) => {
    setSteps((prev) => {
      const exists = prev.find((s) => s.key === key);
      if (exists) {
        return prev.map((s) => (s.key === key ? { ...s, status, message: message || s.message } : s));
      }
      return [...prev, { key, label: STEP_LABELS[key] || key, status, message }];
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) { setError("请输入容器名称"); return; }

    setCreating(true);
    setError("");
    setSteps([]);

    const env: Record<string, string> = {};
    envText.split("\n").filter(Boolean).forEach((line) => {
      const idx = line.indexOf("=");
      if (idx > 0) env[line.slice(0, idx).trim()] = line.slice(idx + 1).trim();
    });

    const body = JSON.stringify({
      name: name.trim(),
      image: image.trim(),
      dataPath: dataPath.trim(),
      port: parseInt(port) || 18789,
      env,
      description: description.trim() || undefined,
    });

    const abort = new AbortController();
    abortRef.current = abort;

    try {
      const res = await fetch(`${API_BASE}/api/create-container`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body,
        signal: abort.signal,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        throw new Error(err.error || res.statusText);
      }

      const contentType = res.headers.get("content-type") || "";
      if (contentType.includes("text/event-stream")) {
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
              if (evt.step === "done" && evt.status === "done") {
                updateStep("done", "done", "容器创建成功");
                setTimeout(() => router.push("/"), 800);
                return;
              }
              updateStep(evt.step, evt.status, evt.message);
              if (evt.status === "error") {
                setError(evt.message);
                setCreating(false);
                return;
              }
            } catch {}
          }
        }
      } else {
        // Fallback: non-SSE response (shouldn't happen but just in case)
        router.push("/");
      }
    } catch (e: any) {
      if (e.name !== "AbortError") {
        setError(e.message);
      }
    }
    setCreating(false);
  };

  const stepIcon = (status: StepInfo["status"]) => {
    switch (status) {
      case "pending": return <span className="text-gray-600">○</span>;
      case "running": return <span className="text-blue-400 animate-pulse">◉</span>;
      case "done": return <span className="text-green-400">✓</span>;
      case "warn": return <span className="text-yellow-400">⚠</span>;
      case "error": return <span className="text-red-400">✗</span>;
    }
  };

  return (
    <div className="max-w-xl">
      <h1 className="text-2xl font-bold mb-6">创建 OpenClaw 容器</h1>
      {error && <div className="mb-4 p-3 bg-red-900/50 border border-red-700 rounded text-red-300 text-sm">{error}</div>}

      {steps.length > 0 && (
        <div className="mb-6 p-4 bg-gray-900/80 border border-gray-700 rounded space-y-2">
          <div className="text-sm text-gray-400 mb-2">创建进度</div>
          {steps.map((s) => (
            <div key={s.key} className="flex items-start gap-2 text-sm">
              <span className="mt-0.5 w-4 text-center flex-shrink-0">{stepIcon(s.status)}</span>
              <div className="min-w-0">
                <span className={s.status === "running" ? "text-blue-300" : s.status === "error" ? "text-red-300" : s.status === "warn" ? "text-yellow-300" : "text-gray-300"}>
                  {s.label}
                </span>
                {s.message && (
                  <span className={`ml-2 text-xs ${s.status === "done" ? "text-gray-600" : "text-gray-500"}`}>{s.message}</span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm text-gray-400 mb-1">容器名称 *</label>
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="my-openclaw" disabled={creating} className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-600 disabled:opacity-50" />
        </div>
        <div>
          <label className="block text-sm text-gray-400 mb-1">Agent 描述</label>
          <textarea value={description} onChange={(e) => setDescription(e.target.value)} placeholder="描述这个 Agent 的用途、性格、专业领域等，AI 将据此生成丰富的配置文件。留空则使用默认模板。" rows={3} disabled={creating} className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-600 resize-none disabled:opacity-50" />
        </div>
        <div>
          <label className="block text-sm text-gray-400 mb-1">镜像</label>
          <input value={image} onChange={(e) => setImage(e.target.value)} disabled={creating} className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-600 disabled:opacity-50" />
        </div>
        <div>
          <label className="block text-sm text-gray-400 mb-1">数据路径</label>
          <input value={dataPath} onChange={(e) => setDataPath(e.target.value)} placeholder={`/home/evan/.openclaw-${name || "name"}`} disabled={creating} className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-600 disabled:opacity-50" />
          <p className="text-xs text-gray-600 mt-1">留空则根据名称自动生成</p>
        </div>
        <div>
          <label className="block text-sm text-gray-400 mb-1">主机端口</label>
          <input value={port} onChange={(e) => setPort(e.target.value)} type="number" disabled={creating} className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-600 disabled:opacity-50" />
        </div>
        <div>
          <label className="block text-sm text-gray-400 mb-1">环境变量</label>
          <textarea value={envText} onChange={(e) => setEnvText(e.target.value)} placeholder={"KEY=value\nANOTHER_KEY=value"} rows={4} disabled={creating} className="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm font-mono focus:outline-none focus:border-blue-600 resize-none disabled:opacity-50" />
          <p className="text-xs text-gray-600 mt-1">每行一个，KEY=value 格式</p>
        </div>
        <div className="flex gap-3 pt-2">
          <button type="submit" disabled={creating} className="px-4 py-2 bg-blue-700 hover:bg-blue-600 disabled:opacity-50 rounded text-sm">
            {creating ? "创建中..." : "创建容器"}
          </button>
          <Link href="/" className={`px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded text-sm ${creating ? "pointer-events-none opacity-50" : ""}`}>取消</Link>
        </div>
      </form>
    </div>
  );
}
