"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api, Conversation, ConversationDetail } from "@/lib/api";
import Link from "next/link";

export default function ConversationsPage() {
  const { id } = useParams<{ id: string }>();
  const [list, setList] = useState<Conversation[]>([]);
  const [detail, setDetail] = useState<ConversationDetail | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api.listConversations(id).then(setList).catch((e) => setError(e.message));
  }, [id]);

  const openConversation = (sid: string) => {
    api.getConversation(id, sid).then(setDetail).catch((e) => setError(e.message));
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">对话记录</h1>
        <Link href={`/containers/${id}`} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">返回</Link>
      </div>
      {error && <div className="mb-3 text-sm text-red-400">{error}</div>}
      <div className="flex flex-col md:flex-row gap-4 h-[calc(100vh-12rem)]">
        <div className="w-full md:w-72 bg-gray-900 border border-gray-800 rounded-lg p-3 overflow-y-auto flex-shrink-0 max-h-48 md:max-h-none">
          {list.length === 0 ? (
            <p className="text-gray-600 text-sm">暂无对话记录</p>
          ) : (
            list.map((c) => (
              <button
                key={c.sessionId}
                onClick={() => openConversation(c.sessionId)}
                className={`w-full text-left px-3 py-2 rounded text-sm hover:bg-gray-800 mb-1 ${detail?.sessionId === c.sessionId ? "bg-gray-800" : ""}`}
              >
                <div className="text-gray-200 truncate">{c.title}</div>
                <div className="text-xs text-gray-500">{c.updatedAt}</div>
              </button>
            ))
          )}
        </div>
        <div className="flex-1 bg-gray-900 border border-gray-800 rounded-lg p-4 overflow-y-auto">
          {detail ? (
            <div className="space-y-4">
              <h2 className="text-lg font-semibold mb-4">{detail.title}</h2>
              {detail.messages?.map((m, i) => (
                <div key={i} className={`flex ${m.role === "user" ? "justify-end" : "justify-start"}`}>
                  <div className={`max-w-[80%] px-4 py-2 rounded-lg text-sm whitespace-pre-wrap ${m.role === "user" ? "bg-blue-900 text-blue-100" : "bg-gray-800 text-gray-200"}`}>
                    <div className="text-xs text-gray-500 mb-1">{m.role}</div>
                    {m.content}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex items-center justify-center h-full text-gray-600">选择一个对话查看</div>
          )}
        </div>
      </div>
    </div>
  );
}
