"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { api, UserPreferences } from "@/lib/api";

export default function SettingsPage() {
  const router = useRouter();
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const [prefs, setPrefs] = useState<UserPreferences>({ username: "", style: "", tools: "", extraContext: "", variables: {} });
  const [prefsSaving, setPrefsSaving] = useState(false);
  const [prefsMessage, setPrefsMessage] = useState("");
  const [prefsError, setPrefsError] = useState("");

  // Variable editing
  const [newVarKey, setNewVarKey] = useState("");
  const [newVarValue, setNewVarValue] = useState("");

  useEffect(() => {
    api.getPreferences().then((p) => setPrefs({ ...p, variables: p.variables || {} })).catch(() => {});
  }, []);

  const addVariable = () => {
    const key = newVarKey.trim().toUpperCase().replace(/[^A-Z0-9_]/g, "_");
    if (!key || !newVarValue.trim()) return;
    setPrefs({ ...prefs, variables: { ...prefs.variables, [key]: newVarValue.trim() } });
    setNewVarKey("");
    setNewVarValue("");
  };

  const removeVariable = (key: string) => {
    const vars = { ...prefs.variables };
    delete vars[key];
    setPrefs({ ...prefs, variables: vars });
  };

  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(""); setMessage("");
    if (!oldPassword.trim() || !newPassword.trim() || !confirmPassword.trim()) { setError("请填写所有字段"); return; }
    if (newPassword !== confirmPassword) { setError("两次输入的新密码不一致"); return; }
    if (newPassword.length < 6) { setError("新密码至少需要 6 个字符"); return; }
    setLoading(true);
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"}/api/auth/password`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, credentials: "include",
        body: JSON.stringify({ oldPassword, newPassword }),
      });
      const data = await res.json();
      if (res.ok) { setMessage("密码修改成功"); setOldPassword(""); setNewPassword(""); setConfirmPassword(""); }
      else { setError(data.error || "密码修改失败"); }
    } catch (e: any) { setError(e.message || "密码修改失败"); }
    setLoading(false);
  };

  const handlePrefsSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setPrefsError(""); setPrefsMessage("");
    setPrefsSaving(true);
    try {
      await api.savePreferences(prefs);
      setPrefsMessage("偏好设置已保存");
    } catch (e: any) { setPrefsError(e.message || "保存失败"); }
    setPrefsSaving(false);
  };

  const inputClass = "w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-600";
  const vars = prefs.variables || {};

  return (
    <div className="max-w-md space-y-6">
      <h1 className="text-2xl font-bold mb-6">设置</h1>

      {/* AI Preferences */}
      <div className="bg-gray-900 border border-gray-800 rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-1">AI 生成偏好</h2>
        <p className="text-xs text-gray-500 mb-4">创建容器时，AI 会根据这些信息生成更贴合你的配置文件</p>

        {prefsError && <div className="mb-4 p-3 bg-red-900/50 border border-red-700 rounded text-red-300 text-sm">{prefsError}</div>}
        {prefsMessage && <div className="mb-4 p-3 bg-green-900/50 border border-green-700 rounded text-green-300 text-sm">{prefsMessage}</div>}

        <form onSubmit={handlePrefsSave} className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">用户名称</label>
            <input value={prefs.username} onChange={(e) => setPrefs({ ...prefs, username: e.target.value })} className={inputClass} placeholder="你的名字或昵称" />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">偏好风格</label>
            <textarea value={prefs.style} onChange={(e) => setPrefs({ ...prefs, style: e.target.value })} className={inputClass + " resize-none"} rows={2} placeholder="例如：简洁专业、幽默轻松、严谨学术..." />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">常用工具</label>
            <textarea value={prefs.tools} onChange={(e) => setPrefs({ ...prefs, tools: e.target.value })} className={inputClass + " resize-none"} rows={3} placeholder={"可使用 {{变量名}} 引用下方定义的变量\n例如：API Key: {{OPENAI_KEY}}"} />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">补充信息</label>
            <textarea value={prefs.extraContext} onChange={(e) => setPrefs({ ...prefs, extraContext: e.target.value })} className={inputClass + " resize-none"} rows={3} placeholder="其他希望 AI 了解的信息，支持 {{变量名}} 引用" />
          </div>

          {/* Variables */}
          <div>
            <label className="block text-sm text-gray-400 mb-2">变量定义</label>
            <p className="text-xs text-gray-600 mb-2">定义敏感信息（如 API 密钥），在上方用 {"{{变量名}}"} 引用。变量值加密存储，不会在页面完整显示。</p>

            {Object.keys(vars).length > 0 && (
              <div className="space-y-2 mb-3">
                {Object.entries(vars).map(([k, v]) => (
                  <div key={k} className="flex items-center gap-2">
                    <span className="text-xs text-blue-400 font-mono min-w-0 shrink-0">{`{{${k}}}`}</span>
                    <span className="text-xs text-gray-500 truncate flex-1 font-mono">{v}</span>
                    <button type="button" onClick={() => removeVariable(k)} className="text-red-500 hover:text-red-400 text-xs shrink-0">删除</button>
                  </div>
                ))}
              </div>
            )}

            <div className="flex gap-2">
              <input value={newVarKey} onChange={(e) => setNewVarKey(e.target.value)} className="w-28 bg-gray-800 border border-gray-700 rounded px-2 py-1.5 text-xs font-mono focus:outline-none focus:border-blue-600" placeholder="变量名" />
              <input value={newVarValue} onChange={(e) => setNewVarValue(e.target.value)} className="flex-1 bg-gray-800 border border-gray-700 rounded px-2 py-1.5 text-xs font-mono focus:outline-none focus:border-blue-600" placeholder="变量值（如 API 密钥）" />
              <button type="button" onClick={addVariable} className="px-3 py-1.5 bg-gray-700 hover:bg-gray-600 rounded text-xs shrink-0">添加</button>
            </div>
          </div>

          <button type="submit" disabled={prefsSaving} className="w-full px-4 py-2 bg-blue-700 hover:bg-blue-600 disabled:opacity-50 rounded text-sm font-medium">
            {prefsSaving ? "保存中..." : "保存偏好"}
          </button>
        </form>
      </div>

      {/* Password */}
      <div className="bg-gray-900 border border-gray-800 rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">修改密码</h2>
        {error && <div className="mb-4 p-3 bg-red-900/50 border border-red-700 rounded text-red-300 text-sm">{error}</div>}
        {message && <div className="mb-4 p-3 bg-green-900/50 border border-green-700 rounded text-green-300 text-sm">{message}</div>}
        <form onSubmit={handlePasswordSubmit} className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">当前密码</label>
            <input type="password" value={oldPassword} onChange={(e) => setOldPassword(e.target.value)} className={inputClass} placeholder="请输入当前密码" autoComplete="current-password" />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">新密码</label>
            <input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} className={inputClass} placeholder="请输入新密码" autoComplete="new-password" />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">确认新密码</label>
            <input type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} className={inputClass} placeholder="请再次输入新密码" autoComplete="new-password" />
          </div>
          <div className="flex gap-3 pt-2">
            <button type="submit" disabled={loading} className="flex-1 px-4 py-2 bg-blue-700 hover:bg-blue-600 disabled:opacity-50 rounded text-sm font-medium">
              {loading ? "修改中..." : "修改密码"}
            </button>
            <button type="button" onClick={() => router.push("/")} className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded text-sm">返回</button>
          </div>
        </form>
      </div>
    </div>
  );
}
