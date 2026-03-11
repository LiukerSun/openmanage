"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams } from "next/navigation";
import { api, FileEntry, FileContent } from "@/lib/api";
import { CodeEditor } from "@/components/CodeEditor";
import Link from "next/link";

export default function FilesPage() {
  const { id } = useParams<{ id: string }>();
  const [entries, setEntries] = useState<FileEntry[]>([]);
  const [currentPath, setCurrentPath] = useState("");
  const [file, setFile] = useState<FileContent | null>(null);
  const [edited, setEdited] = useState("");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");

  const loadDir = useCallback((path: string) => {
    setFile(null);
    setCurrentPath(path);
    const req = path ? api.readDir(id, path) : api.listFiles(id);
    req.then(setEntries).catch((e) => setMessage(e.message));
  }, [id]);

  useEffect(() => { loadDir(""); }, [loadDir]);

  const openFile = (entry: FileEntry) => {
    if (entry.isDir) {
      loadDir(entry.path);
    } else {
      api.readFile(id, entry.path).then((f) => { setFile(f); setEdited(f.content); }).catch((e) => setMessage(e.message));
    }
  };

  const save = async () => {
    if (!file) return;
    setSaving(true);
    try {
      await api.writeFile(id, file.path, edited);
      setMessage("已保存");
      setTimeout(() => setMessage(""), 2000);
    } catch (e: any) { setMessage(e.message); }
    setSaving(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "s") { e.preventDefault(); save(); }
  };

  const goUp = () => {
    const parts = currentPath.split("/").filter(Boolean);
    parts.pop();
    loadDir(parts.join("/"));
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">文件</h1>
        <Link href={`/containers/${id}`} className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">返回</Link>
      </div>
      {message && <div className="mb-3 text-sm text-yellow-400">{message}</div>}
      <div className="flex flex-col md:flex-row gap-4 h-[calc(100vh-12rem)]">
        <div className="w-full md:w-64 bg-gray-900 border border-gray-800 rounded-lg p-3 overflow-y-auto flex-shrink-0 max-h-48 md:max-h-none">
          <div className="text-xs text-gray-500 mb-2">/{currentPath}</div>
          {currentPath && (
            <button onClick={goUp} className="w-full text-left px-2 py-1 text-sm text-gray-400 hover:bg-gray-800 rounded">..</button>
          )}
          {entries.map((e) => (
            <button key={e.path} onClick={() => openFile(e)} className={`w-full text-left px-2 py-1 text-sm rounded hover:bg-gray-800 ${e.isDir ? "text-blue-400" : "text-gray-300"} ${file?.path === e.path ? "bg-gray-800" : ""}`}>
              {e.isDir ? "📁 " : "📄 "}{e.name}
            </button>
          ))}
        </div>
        <div className="flex-1 flex flex-col">
          {file ? (
            <>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-400">{file.path}</span>
                <button onClick={save} disabled={saving} className="px-3 py-1 bg-blue-700 hover:bg-blue-600 disabled:opacity-50 rounded text-sm">
                  {saving ? "保存中..." : "保存 (Ctrl+S)"}
                </button>
              </div>
              <CodeEditor
                value={edited}
                onChange={setEdited}
                onKeyDown={handleKeyDown}
                filePath={file.path}
              />
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-gray-600">选择一个文件进行编辑</div>
          )}
        </div>
      </div>
    </div>
  );
}
