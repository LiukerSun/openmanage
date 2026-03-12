"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api, ContainerInfo } from "@/lib/api";
import { useContainerStats } from "@/lib/ws";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import Link from "next/link";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return (bytes / Math.pow(1024, i)).toFixed(1) + " " + units[i];
}

function StatCard({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="bg-gray-800 rounded-lg p-3">
      <div className="text-xs text-gray-500 mb-1">{label}</div>
      <div className="text-lg font-semibold">{value}</div>
      {sub && <div className="text-xs text-gray-500 mt-1">{sub}</div>}
    </div>
  );
}

export default function ContainerDetail() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [container, setContainer] = useState<ContainerInfo | null>(null);
  const stats = useContainerStats(container?.state === "running" ? id : null);
  const [error, setError] = useState("");
  const [showDelete, setShowDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [editing, setEditing] = useState(false);
  const [newName, setNewName] = useState("");
  const [renaming, setRenaming] = useState(false);

  useEffect(() => {
    api.getContainer(id).then((c) => { setContainer(c); setNewName(c.name || ""); }).catch((e) => setError(e.message));
  }, [id]);

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await api.deleteContainer(id);
      router.push("/");
    } catch (e: any) { setError(e.message); }
    setDeleting(false);
    setShowDelete(false);
  };

  const handleRename = async () => {
    if (!newName.trim() || newName === container?.name) { setEditing(false); return; }
    setRenaming(true);
    try {
      await api.updateContainer(id, { name: newName.trim() });
      const updated = await api.getContainer(id);
      setContainer(updated);
      setEditing(false);
    } catch (e: any) { setError(e.message); }
    setRenaming(false);
  };

  if (error) return <div className="text-red-400">错误: {error}</div>;
  if (!container) return <div className="text-gray-400">加载中...</div>;

  return (
    <div>
      <div className="flex items-center gap-3 mb-4">
        {editing ? (
          <>
            <input value={newName} onChange={(e) => setNewName(e.target.value)} onKeyDown={(e) => e.key === "Enter" && handleRename()} className="text-2xl font-bold bg-gray-800 border border-gray-600 rounded px-2 py-1" autoFocus />
            <button onClick={handleRename} disabled={renaming} className="px-3 py-1 bg-blue-700 hover:bg-blue-600 rounded text-sm">{renaming ? "保存中..." : "保存"}</button>
            <button onClick={() => { setEditing(false); setNewName(container.name || ""); }} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">取消</button>
          </>
        ) : (
          <>
            <h1 className="text-2xl font-bold">{container.name || container.id}</h1>
            <button onClick={() => setEditing(true)} className="px-2 py-1 bg-gray-700 hover:bg-gray-600 rounded text-xs">重命名</button>
          </>
        )}
      </div>
      <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 mb-6">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
          <div><span className="text-gray-500">容器 ID:</span> {container.id}</div>
          <div><span className="text-gray-500">镜像:</span> {container.image}</div>
          <div><span className="text-gray-500">状态:</span> <span className={container.state === "running" ? "text-green-400" : "text-red-400"}>{container.state === "running" ? "运行中" : container.state === "exited" ? "已停止" : container.state}</span></div>
          <div><span className="text-gray-500">创建时间:</span> {new Date(container.created * 1000).toLocaleString("zh-CN")}</div>
        </div>
        {container.mounts.length > 0 && (
          <div className="mt-4">
            <h3 className="text-sm text-gray-500 mb-2">挂载卷</h3>
            {container.mounts.map((m, i) => (
              <div key={i} className="text-xs text-gray-400">{m.source} → {m.destination} {m.rw ? "(读写)" : "(只读)"}</div>
            ))}
          </div>
        )}
      </div>

      {container.state === "running" && stats && (
        <div className="mb-6">
          <h2 className="text-lg font-semibold mb-3">资源监控</h2>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <StatCard label="CPU" value={stats.cpuPercent.toFixed(1) + "%"} />
            <StatCard label="内存" value={formatBytes(stats.memUsage)} sub={stats.memPercent.toFixed(1) + "% / " + formatBytes(stats.memLimit)} />
            <StatCard label="网络接收" value={formatBytes(stats.netRx)} />
            <StatCard label="网络发送" value={formatBytes(stats.netTx)} />
          </div>
          <div className="mt-3 text-xs text-gray-600">实时推送 | 进程数: {stats.pids}</div>
        </div>
      )}

      <div className="flex flex-wrap gap-3">
        <Link href={`/containers/${id}/logs`} className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded">日志</Link>
        <Link href={`/containers/${id}/files`} className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded">文件</Link>
        <Link href={`/containers/${id}/conversations`} className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded">对话</Link>
        <Link href={`/containers/${id}/forum`} className="px-4 py-2 bg-indigo-700 hover:bg-indigo-600 rounded">论坛</Link>
        <Link href={`/containers/${id}/cron`} className="px-4 py-2 bg-teal-700 hover:bg-teal-600 rounded">定时任务</Link>
        <button onClick={() => setShowDelete(true)} className="px-4 py-2 bg-red-900 hover:bg-red-800 rounded">删除容器</button>
      </div>
      <ConfirmDialog
        open={showDelete}
        title="删除容器"
        message={<>确定要删除容器 <span className="text-white font-medium">{container.name || container.id}</span> 吗？此操作不可撤销。</>}
        confirmLabel="删除"
        variant="danger"
        loading={deleting}
        onConfirm={handleDelete}
        onCancel={() => setShowDelete(false)}
      />
    </div>
  );
}
