"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { api, ForumActivity } from "@/lib/api";
import Link from "next/link";

export default function ForumActivityPage() {
  const params = useParams();
  const id = params.id as string;
  const [activity, setActivity] = useState<ForumActivity | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    api.getForumActivity(id)
      .then(setActivity)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) return <div className="text-gray-400">加载中...</div>;
  if (error) return (
    <div>
      <div className="text-red-400 mb-4">错误: {error}</div>
      <Link href={`/containers/${id}`} className="text-blue-400 hover:underline text-sm">返回容器详情</Link>
    </div>
  );
  if (!activity) return null;

  const stats = [
    { label: "发帖", value: activity.topicCount, color: "text-blue-400" },
    { label: "回帖", value: activity.postCount, color: "text-green-400" },
    { label: "获赞", value: activity.likesReceived, color: "text-yellow-400" },
    { label: "点赞", value: activity.likesGiven, color: "text-pink-400" },
    { label: "访问天数", value: activity.daysVisited, color: "text-purple-400" },
  ];

  return (
    <div>
      <div className="flex items-center gap-2 mb-6">
        <Link href={`/containers/${id}`} className="text-gray-400 hover:text-white text-sm">← 容器详情</Link>
        <span className="text-gray-600">/</span>
        <h1 className="text-xl font-bold">论坛活动</h1>
        <span className="text-sm text-gray-500 font-mono">@{activity.username}</span>
      </div>

      <div className="grid grid-cols-5 gap-3 mb-6">
        {stats.map((s) => (
          <div key={s.label} className="bg-gray-900 border border-gray-800 rounded-lg p-3 text-center">
            <div className={`text-2xl font-bold ${s.color}`}>{s.value}</div>
            <div className="text-xs text-gray-500 mt-1">{s.label}</div>
          </div>
        ))}
      </div>

      <h2 className="text-lg font-semibold mb-3">最近活动</h2>
      {activity.actions.length === 0 ? (
        <p className="text-gray-500 text-sm">暂无论坛活动记录</p>
      ) : (
        <div className="space-y-2">
          {activity.actions.map((a, i) => (
            <div key={i} className="bg-gray-900 border border-gray-800 rounded p-3">
              <div className="flex items-center gap-2 mb-1">
                <span className={`text-xs px-1.5 py-0.5 rounded ${a.type === "topic" ? "bg-blue-900 text-blue-300" : "bg-green-900 text-green-300"}`}>
                  {a.type === "topic" ? "发帖" : "回帖"}
                </span>
                <span className="text-sm text-gray-300 truncate">{a.title}</span>
              </div>
              {a.excerpt && (
                <p className="text-xs text-gray-500 line-clamp-2" dangerouslySetInnerHTML={{ __html: a.excerpt }} />
              )}
              <div className="text-xs text-gray-600 mt-1">
                {new Date(a.createdAt).toLocaleString("zh-CN")}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
