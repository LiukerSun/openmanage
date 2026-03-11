"use client";

import { useState } from "react";
import { api } from "@/lib/api";
import { useContainerList } from "@/lib/ws";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import Link from "next/link";

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

export default function Dashboard() {
  const { containers, loading } = useContainerList();
  const [error, setError] = useState("");
  const [confirm, setConfirm] = useState<{ id: string; name: string; act: "start" | "stop" | "restart" | "delete" } | null>(null);
  const [acting, setActing] = useState(false);

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

  if (loading) return <div className="text-gray-400">加载中...</div>;
  if (error) return <div className="text-red-400">错误: {error}</div>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">OpenClaw 容器</h1>
        <Link href="/create" className="px-4 py-2 bg-blue-700 hover:bg-blue-600 rounded text-sm">+ 新建容器</Link>
      </div>
      {containers.length === 0 ? (
        <p className="text-gray-500">未找到 OpenClaw 容器。请确保容器带有 <code className="bg-gray-800 px-1 rounded">openmanage.openclaw=true</code> 标签。</p>
      ) : (
        <div className="grid gap-4">
          {containers.map((c) => (
            <div key={c.id} className="bg-gray-900 border border-gray-800 rounded-lg p-4">
              <div className="flex items-center justify-between mb-3">
                <div>
                  <Link href={`/containers/${c.id}`} className="text-lg font-semibold hover:text-blue-400">{c.name || c.id}</Link>
                  <p className="text-sm text-gray-500">{c.image}</p>
                </div>
                <span className={`px-2 py-1 rounded text-xs font-medium ${c.state === "running" ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300"}`}>
                  {c.state === "running" ? "运行中" : c.state === "exited" ? "已停止" : c.state}
                </span>
              </div>
              <div className="flex flex-wrap gap-2">
                <button onClick={() => requestAction(c.id, c.name || c.id, "start")} disabled={c.state === "running"} className="px-3 py-1 bg-green-700 hover:bg-green-600 disabled:opacity-30 rounded text-sm">启动</button>
                <button onClick={() => requestAction(c.id, c.name || c.id, "stop")} disabled={c.state !== "running"} className="px-3 py-1 bg-red-700 hover:bg-red-600 disabled:opacity-30 rounded text-sm">停止</button>
                <button onClick={() => requestAction(c.id, c.name || c.id, "restart")} disabled={c.state !== "running"} className="px-3 py-1 bg-yellow-700 hover:bg-yellow-600 disabled:opacity-30 rounded text-sm">重启</button>
                <button onClick={() => requestAction(c.id, c.name || c.id, "delete")} className="px-3 py-1 bg-red-900 hover:bg-red-800 rounded text-sm">删除</button>
                <Link href={`/containers/${c.id}/logs`} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">日志</Link>
                <Link href={`/containers/${c.id}/files`} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">文件</Link>
                <Link href={`/containers/${c.id}/conversations`} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">对话</Link>
              </div>
            </div>
          ))}
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
    </div>
  );
}
