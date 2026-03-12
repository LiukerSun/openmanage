"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams } from "next/navigation";
import { api, CronJob, HeartbeatConfig } from "@/lib/api";

export default function CronPage() {
  const { id } = useParams<{ id: string }>();
  const [jobs, setJobs] = useState<CronJob[]>([]);
  const [heartbeat, setHeartbeat] = useState<HeartbeatConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Heartbeat edit
  const [editingHB, setEditingHB] = useState(false);
  const [hbValue, setHbValue] = useState("");
  const [savingHB, setSavingHB] = useState(false);

  // New cron form
  const [showForm, setShowForm] = useState(false);
  const [newName, setNewName] = useState("");
  const [newSchedule, setNewSchedule] = useState("");
  const [newPrompt, setNewPrompt] = useState("");
  const [adding, setAdding] = useState(false);

  // AI generate
  const [aiDesc, setAiDesc] = useState("");
  const [generating, setGenerating] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const data = await api.listCronJobs(id);
      setJobs(data.jobs || []);
      setHeartbeat(data.heartbeat || null);
      setHbValue(data.heartbeat?.every || "");
    } catch (e: any) {
      setError(e.message);
    }
    setLoading(false);
  }, [id]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleSaveHeartbeat = async () => {
    setSavingHB(true);
    try {
      await api.updateHeartbeat(id, hbValue);
      setHeartbeat(hbValue ? { every: hbValue } : null);
      setEditingHB(false);
    } catch (e: any) { setError(e.message); }
    setSavingHB(false);
  };

  const handleAddCron = async () => {
    if (!newSchedule.trim() || !newPrompt.trim()) return;
    setAdding(true);
    try {
      await api.addCronJob(id, newName.trim() || newSchedule, newSchedule, newPrompt);
      setNewName("");
      setNewSchedule("");
      setNewPrompt("");
      setAiDesc("");
      setShowForm(false);
      await fetchData();
    } catch (e: any) { setError(e.message); }
    setAdding(false);
  };

  const handleAIGenerate = async () => {
    if (!aiDesc.trim()) return;
    setGenerating(true);
    setError("");
    try {
      const result = await api.generateCronJob(id, aiDesc);
      setNewSchedule(result.schedule);
      setNewPrompt(result.prompt);
    } catch (e: any) { setError(e.message); }
    setGenerating(false);
  };

  const handleToggle = async (jobId: string, enabled: boolean) => {
    try {
      await api.toggleCronJob(id, jobId, enabled);
      await fetchData();
    } catch (e: any) { setError(e.message); }
  };

  const handleRun = async (jobId: string) => {
    try {
      await api.runCronJob(id, jobId);
    } catch (e: any) { setError(e.message); }
  };

  const handleRemove = async (jobId: string) => {
    try {
      await api.removeCronJob(id, jobId);
      await fetchData();
    } catch (e: any) { setError(e.message); }
  };

  if (loading) return <div className="text-gray-400">加载中...</div>;

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">定时任务管理</h1>
      {error && <div className="text-red-400 mb-4">{error}</div>}

      {/* Heartbeat Section */}
      <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 mb-6">
        <div className="flex items-center justify-between mb-2">
          <h2 className="text-lg font-semibold">Heartbeat 心跳</h2>
          {!editingHB && (
            <button onClick={() => setEditingHB(true)} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">
              编辑
            </button>
          )}
        </div>
        {editingHB ? (
          <div className="flex items-center gap-3">
            <input
              value={hbValue}
              onChange={(e) => setHbValue(e.target.value)}
              placeholder="如 30m, 1h, 2h（留空禁用）"
              className="flex-1 bg-gray-800 border border-gray-600 rounded px-3 py-2 text-sm"
            />
            <button onClick={handleSaveHeartbeat} disabled={savingHB} className="px-3 py-1 bg-blue-700 hover:bg-blue-600 rounded text-sm">
              {savingHB ? "保存中..." : "保存"}
            </button>
            <button onClick={() => { setEditingHB(false); setHbValue(heartbeat?.every || ""); }} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">
              取消
            </button>
          </div>
        ) : (
          <div className="text-sm text-gray-400">
            {heartbeat?.every ? (
              <span>间隔: <span className="text-green-400 font-mono">{heartbeat.every}</span></span>
            ) : (
              <span className="text-yellow-500">未配置（使用默认 30m 或已禁用）</span>
            )}
          </div>
        )}
        <p className="text-xs text-gray-600 mt-2">Heartbeat 行为由 workspace/HEARTBEAT.md 定义，修改间隔后需重启容器生效。</p>
      </div>

      {/* Cron Jobs Section */}
      <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">Cron 定时任务</h2>
          <button onClick={() => setShowForm(!showForm)} className="px-3 py-1 bg-teal-700 hover:bg-teal-600 rounded text-sm">
            {showForm ? "取消" : "新建任务"}
          </button>
        </div>

        {showForm && (
          <div className="bg-gray-800 rounded-lg p-4 mb-4 space-y-3">
            <div className="bg-gray-750 border border-teal-800 rounded-lg p-3 space-y-2">
              <label className="block text-xs text-teal-400 mb-1">AI 辅助 — 用自然语言描述任务</label>
              <div className="flex gap-2">
                <input
                  value={aiDesc}
                  onChange={(e) => setAiDesc(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handleAIGenerate()}
                  placeholder="如：每小时检查论坛新帖并回复、每天早上发一条问候帖"
                  className="flex-1 bg-gray-900 border border-gray-600 rounded px-3 py-2 text-sm"
                />
                <button onClick={handleAIGenerate} disabled={generating || !aiDesc.trim()} className="px-4 py-2 bg-teal-700 hover:bg-teal-600 disabled:opacity-50 rounded text-sm whitespace-nowrap">
                  {generating ? "生成中..." : "AI 生成"}
                </button>
              </div>
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">任务名称</label>
              <input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="如 check-forum（可选，留空自动生成）"
                className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">Cron 表达式</label>
              <input
                value={newSchedule}
                onChange={(e) => setNewSchedule(e.target.value)}
                placeholder="如 */30 * * * *（每30分钟）"
                className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2 text-sm font-mono"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">执行 Prompt</label>
              <textarea
                value={newPrompt}
                onChange={(e) => setNewPrompt(e.target.value)}
                placeholder="Agent 收到的指令内容..."
                rows={3}
                className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2 text-sm"
              />
            </div>
            <button onClick={handleAddCron} disabled={adding} className="px-4 py-2 bg-teal-700 hover:bg-teal-600 rounded text-sm">
              {adding ? "创建中..." : "创建"}
            </button>
          </div>
        )}

        {jobs.length === 0 ? (
          <div className="text-sm text-gray-500 py-4 text-center">暂无 cron 任务</div>
        ) : (
          <div className="space-y-3">
            {jobs.map((job) => (
              <div key={job.id} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className={`inline-block w-2 h-2 rounded-full ${job.enabled ? "bg-green-400" : "bg-gray-600"}`} />
                    <span className="font-mono text-sm">{job.schedule}</span>
                    {job.name && <span className="text-xs text-gray-500">({job.name})</span>}
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleToggle(job.id, !job.enabled)}
                      className={`px-2 py-1 rounded text-xs ${job.enabled ? "bg-yellow-800 hover:bg-yellow-700" : "bg-green-800 hover:bg-green-700"}`}
                    >
                      {job.enabled ? "禁用" : "启用"}
                    </button>
                    <button onClick={() => handleRun(job.id)} className="px-2 py-1 bg-blue-800 hover:bg-blue-700 rounded text-xs">
                      立即执行
                    </button>
                    <button onClick={() => handleRemove(job.id)} className="px-2 py-1 bg-red-900 hover:bg-red-800 rounded text-xs">
                      删除
                    </button>
                  </div>
                </div>
                <div className="text-xs text-gray-400 truncate">{job.prompt}</div>
                {(job.lastRun || job.nextRun) && (
                  <div className="flex gap-4 mt-1 text-xs text-gray-600">
                    {job.lastRun && <span>上次: {job.lastRun}</span>}
                    {job.nextRun && <span>下次: {job.nextRun}</span>}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
