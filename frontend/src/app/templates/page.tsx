"use client";

import { useEffect, useState, useCallback } from "react";
import { api, FileEntry } from "@/lib/api";
import { CodeEditor } from "@/components/CodeEditor";

export default function TemplatesPage() {
  const [files, setFiles] = useState<FileEntry[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [content, setContent] = useState("");
  const [edited, setEdited] = useState("");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [newName, setNewName] = useState("");
  const [showNew, setShowNew] = useState(false);

  const load = useCallback(() => {
    api.listTemplates().then(setFiles).catch((e) => setMessage(e.message));
  }, []);

  useEffect(() => { load(); }, [load]);

  const openFile = (name: string) => {
    api.readTemplate(name).then((f) => {
      setSelected(name);
      setContent(f.content);
      setEdited(f.content);
    }).catch((e) => setMessage(e.message));
  };

  const save = async () => {
    if (!selected) return;
    setSaving(true);
    try {
      await api.writeTemplate(selected, edited);
      setContent(edited);
      setMessage("已保存");
      setTimeout(() => setMessage(""), 2000);
    } catch (e: any) { setMessage(e.message); }
    setSaving(false);
  };

  const createFile = async () => {
    if (!newName.trim()) return;
    try {
      await api.createTemplate(newName.trim(), "");
      setShowNew(false);
      setNewName("");
      load();
      openFile(newName.trim());
    } catch (e: any) { setMessage(e.message); }
  };

  const deleteFile = async (name: string) => {
    try {
      await api.deleteTemplate(name);
      if (selected === name) { setSelected(null); setContent(""); setEdited(""); }
      load();
    } catch (e: any) { setMessage(e.message); }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "s") { e.preventDefault(); save(); }
  };

  const hasChanges = content !== edited;

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">模板管理</h1>
        <p className="text-sm text-gray-500">使用 {"{{NAME}}"} 作为占位符，创建容器时自动替换</p>
      </div>
      {message && <div className="mb-3 text-sm text-yellow-400">{message}</div>}
      <div className="flex flex-col md:flex-row gap-4 h-[calc(100vh-12rem)]">
        <div className="w-full md:w-64 bg-gray-900 border border-gray-800 rounded-lg p-3 overflow-y-auto flex-shrink-0 max-h-48 md:max-h-none">
          <button onClick={() => setShowNew(!showNew)} className="w-full px-3 py-2 mb-2 bg-blue-700 hover:bg-blue-600 rounded text-sm">+ 新建模板</button>
          {showNew && (
            <div className="mb-2 flex gap-1">
              <input value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="filename.md" className="flex-1 bg-gray-800 border border-gray-700 rounded px-2 py-1 text-sm" onKeyDown={(e) => e.key === "Enter" && createFile()} />
              <button onClick={createFile} className="px-2 py-1 bg-green-700 rounded text-sm">OK</button>
            </div>
          )}
          {files.map((f) => (
            <div key={f.name} className="flex items-center group">
              <button onClick={() => openFile(f.name)} className={`flex-1 text-left px-2 py-1 text-sm rounded hover:bg-gray-800 truncate ${selected === f.name ? "bg-gray-800 text-white" : "text-gray-300"}`}>
                {f.name}
              </button>
              <button onClick={() => deleteFile(f.name)} className="px-1 text-red-500 opacity-0 group-hover:opacity-100 text-xs" title="删除">x</button>
            </div>
          ))}
        </div>
        <div className="flex-1 flex flex-col">
          {selected ? (
            <>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-400">{selected} {hasChanges && <span className="text-yellow-500">(未保存)</span>}</span>
                <button onClick={save} disabled={saving || !hasChanges} className="px-3 py-1 bg-blue-700 hover:bg-blue-600 disabled:opacity-50 rounded text-sm">
                  {saving ? "保存中..." : "保存 (Ctrl+S)"}
                </button>
              </div>
              <CodeEditor
                value={edited}
                onChange={setEdited}
                onKeyDown={handleKeyDown}
              />
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-gray-600">选择一个模板进行编辑</div>
          )}
        </div>
      </div>
    </div>
  );
}
