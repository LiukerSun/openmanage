"use client";

import { useEffect, useState, useRef, useCallback } from "react";
import { useParams } from "next/navigation";
import { api, Conversation, ConversationDetail, Message } from "@/lib/api";
import Link from "next/link";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export default function ConversationsPage() {
  const { id } = useParams<{ id: string }>();
  const [list, setList] = useState<Conversation[]>([]);
  const [detail, setDetail] = useState<ConversationDetail | null>(null);
  const [error, setError] = useState("");
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [statusMsg, setStatusMsg] = useState("");
  const [lastResponseId, setLastResponseId] = useState<string>("");
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Read post shortcut
  const [showReadPost, setShowReadPost] = useState(false);
  const [topicInput, setTopicInput] = useState("");

  const refreshList = useCallback(() => {
    api.listConversations(id).then(setList).catch((e) => setError(e.message));
  }, [id]);

  useEffect(() => {
    refreshList();
  }, [refreshList]);

  // Poll: refresh conversation list every 10s
  useEffect(() => {
    const timer = setInterval(refreshList, 10000);
    return () => clearInterval(timer);
  }, [refreshList]);

  // Poll: refresh current conversation messages every 5s
  useEffect(() => {
    if (!detail?.sessionId) return;
    const sid = detail.sessionId;
    const timer = setInterval(() => {
      api.getConversation(id, sid).then((fresh) => {
        setDetail((prev) => {
          if (!prev || prev.sessionId !== sid) return prev;
          // Only update if message count changed (avoid flicker)
          if (fresh.messages?.length !== prev.messages?.length) {
            return fresh;
          }
          return prev;
        });
      }).catch(() => {});
    }, 5000);
    return () => clearInterval(timer);
  }, [id, detail?.sessionId]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [detail?.messages]);

  const openConversation = (sid: string) => {
    setLastResponseId("");
    api.getConversation(id, sid).then(setDetail).catch((e) => setError(e.message));
  };

  const startNewConversation = () => {
    setDetail(null);
    setInput("");
    setStatusMsg("");
    setLastResponseId("");
  };

  const sendMessage = async () => {
    const msg = input.trim();
    if (!msg || sending) return;

    setSending(true);
    setError("");
    setStatusMsg("正在发送...");

    // Optimistically add user message to the UI
    const userMsg: Message = { role: "user", content: msg };
    if (detail) {
      setDetail({ ...detail, messages: [...(detail.messages || []), userMsg] });
    } else {
      setDetail({
        sessionId: "",
        title: msg.length > 50 ? msg.slice(0, 50) + "..." : msg,
        messages: [userMsg],
      });
    }
    setInput("");

    try {
      const res = await fetch(`${API_BASE}/api/containers/${id}/chat`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ message: msg, previousResponseId: lastResponseId || undefined }),
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
            if (evt.status === "sending") {
              setStatusMsg("Agent 正在思考...");
            } else if (evt.status === "done" && evt.reply) {
              if (evt.responseId) {
                setLastResponseId(evt.responseId);
              }
              const assistantMsg: Message = { role: "assistant", content: evt.reply };
              setDetail((prev) => {
                if (!prev) return prev;
                return { ...prev, messages: [...(prev.messages || []), assistantMsg] };
              });
              setStatusMsg("");
            } else if (evt.status === "error") {
              setError(evt.message || "发送失败");
              setStatusMsg("");
            }
          } catch {
            // ignore parse errors
          }
        }
      }

      // Refresh conversation list to show the new/updated conversation
      setTimeout(refreshList, 1000);
    } catch (e: unknown) {
      const errMsg = e instanceof Error ? e.message : "发送失败";
      setError(errMsg);
      setStatusMsg("");
    } finally {
      setSending(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  const sendReadPost = () => {
    const topic = topicInput.trim();
    if (!topic) return;
    const prompt = `请阅读这个论坛帖子并发表你的看法：${topic}\n\n要求：\n1. 先用工具读取帖子内容\n2. 理解帖子的主题和讨论内容\n3. 发表一条有见地的回复`;
    setInput(prompt);
    setShowReadPost(false);
    setTopicInput("");
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">对话</h1>
        <div className="flex gap-2">
          <button
            onClick={startNewConversation}
            className="px-3 py-1 bg-blue-700 hover:bg-blue-600 rounded text-sm"
          >
            新建对话
          </button>
          <Link href={`/containers/${id}`} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">
            返回
          </Link>
        </div>
      </div>
      {error && <div className="mb-3 text-sm text-red-400">{error}</div>}
      <div className="flex flex-col md:flex-row gap-4 h-[calc(100vh-12rem)]">
        {/* Left: conversation list */}
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

        {/* Right: messages + input */}
        <div className="flex-1 bg-gray-900 border border-gray-800 rounded-lg flex flex-col overflow-hidden">
          {/* Messages area */}
          <div className="flex-1 p-4 overflow-y-auto">
            {detail ? (
              <div className="space-y-4">
                {detail.sessionId && (
                  <h2 className="text-lg font-semibold mb-4">{detail.title}</h2>
                )}
                {detail.messages?.map((m, i) => (
                  <div key={i} className={`flex ${m.role === "user" ? "justify-end" : "justify-start"}`}>
                    <div className={`max-w-[80%] px-4 py-2 rounded-lg text-sm whitespace-pre-wrap ${m.role === "user" ? "bg-blue-900 text-blue-100" : "bg-gray-800 text-gray-200"}`}>
                      <div className="text-xs text-gray-500 mb-1">{m.role}</div>
                      {m.content}
                    </div>
                  </div>
                ))}
                {statusMsg && (
                  <div className="flex justify-start">
                    <div className="px-4 py-2 rounded-lg text-sm bg-gray-800 text-gray-400 animate-pulse">
                      {statusMsg}
                    </div>
                  </div>
                )}
                <div ref={messagesEndRef} />
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-gray-600">
                选择一个对话或新建对话开始聊天
              </div>
            )}
          </div>

          {/* Input area */}
          <div className="border-t border-gray-800 p-3">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-xs text-gray-600">快捷指令:</span>
              <button
                onClick={() => setShowReadPost(!showReadPost)}
                className="px-2 py-0.5 text-xs bg-orange-900 hover:bg-orange-800 text-orange-300 rounded"
              >
                {showReadPost ? "取消" : "阅读帖子"}
              </button>
            </div>
            {showReadPost && (
              <div className="flex gap-2 mb-2">
                <input
                  value={topicInput}
                  onChange={(e) => setTopicInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && sendReadPost()}
                  placeholder="输入帖子 URL 或 Topic ID..."
                  className="flex-1 bg-gray-800 border border-orange-700 rounded px-3 py-1.5 text-sm focus:outline-none focus:border-orange-500"
                  autoFocus
                />
                <button
                  onClick={sendReadPost}
                  disabled={!topicInput.trim()}
                  className="px-3 py-1.5 bg-orange-700 hover:bg-orange-600 disabled:bg-gray-700 disabled:text-gray-500 rounded text-sm"
                >
                  生成
                </button>
              </div>
            )}
            <div className="flex gap-2">
              <textarea
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="输入消息，Enter 发送，Shift+Enter 换行"
                rows={1}
                className="flex-1 bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm resize-none focus:outline-none focus:border-blue-600"
                disabled={sending}
              />
              <button
                onClick={sendMessage}
                disabled={sending || !input.trim()}
                className="px-4 py-2 bg-blue-700 hover:bg-blue-600 disabled:bg-gray-700 disabled:text-gray-500 rounded text-sm whitespace-nowrap"
              >
                {sending ? "发送中..." : "发送"}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
