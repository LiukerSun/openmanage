"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import Link from "next/link";

export default function LogsPage() {
  const { id } = useParams<{ id: string }>();
  const [lines, setLines] = useState<string[]>([]);
  const [following, setFollowing] = useState(true);
  const [connected, setConnected] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  const connect = useCallback(() => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    const url = api.logsURL(id, "200", true);
    fetch(url, {
      credentials: "include",
      signal: controller.signal,
    })
      .then((res) => {
        if (!res.ok || !res.body) {
          setConnected(false);
          return;
        }
        setConnected(true);
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        function read(): Promise<void> {
          return reader.read().then(({ done, value }) => {
            if (done) {
              setConnected(false);
              return;
            }
            buffer += decoder.decode(value, { stream: true });
            const parts = buffer.split("\n");
            buffer = parts.pop() || "";

            const newLines: string[] = [];
            for (const part of parts) {
              if (part.startsWith("data: ")) {
                newLines.push(part.slice(6));
              }
            }
            if (newLines.length > 0) {
              setLines((prev) => [...prev.slice(-2000), ...newLines]);
            }
            return read();
          });
        }

        return read();
      })
      .catch((err) => {
        if (err.name !== "AbortError") {
          setConnected(false);
        }
      });
  }, [id]);

  useEffect(() => {
    connect();
    return () => abortRef.current?.abort();
  }, [connect]);

  useEffect(() => {
    if (following) bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines, following]);

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">日志</h1>
          <span className={`w-2 h-2 rounded-full ${connected ? "bg-green-500" : "bg-red-500"}`} title={connected ? "已连接" : "未连接"} />
        </div>
        <div className="flex gap-2">
          <button onClick={() => setFollowing(!following)} className={`px-3 py-1 rounded text-sm ${following ? "bg-green-700" : "bg-gray-700"}`}>
            {following ? "跟随中" : "已暂停"}
          </button>
          <button onClick={() => setLines([])} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">清空</button>
          {!connected && (
            <button onClick={connect} className="px-3 py-1 bg-blue-700 hover:bg-blue-600 rounded text-sm">重连</button>
          )}
          <Link href={`/containers/${id}`} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">返回</Link>
        </div>
      </div>
      <div className="bg-black rounded-lg p-4 font-mono text-xs h-[calc(100vh-12rem)] overflow-y-auto">
        {lines.length === 0 && (
          <div className="text-gray-600">{connected ? "等待日志输出..." : "未连接到日志流"}</div>
        )}
        {lines.map((line, i) => (
          <div key={i} className="text-gray-300 hover:bg-gray-900">{line}</div>
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
