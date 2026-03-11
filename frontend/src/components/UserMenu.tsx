"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { logoutSimple, getUser, isAuthenticated } from "@/lib/auth";

export function UserMenu() {
  const router = useRouter();
  const [user, setUser] = useState<{ username: string } | null>(null);
  const [showMenu, setShowMenu] = useState(false);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
    if (isAuthenticated()) {
      setUser(getUser());
    }
  }, []);

  const handleLogout = async () => {
    await logoutSimple();
    document.cookie = "auth_token=; path=/; max-age=0";
    router.push("/login");
  };

  const goToSettings = () => {
    setShowMenu(false);
    router.push("/settings");
  };

  if (!mounted) {
    return <div className="border-t border-gray-800 pt-3" />;
  }

  if (!user) {
    return (
      <div className="border-t border-gray-800 pt-3">
        <button
          onClick={() => router.push("/login")}
          className="w-full px-3 py-2 rounded hover:bg-gray-800 text-gray-400 hover:text-white text-sm text-left flex items-center gap-2"
        >
          <span className="inline-flex rounded-full h-2 w-2 bg-gray-500 shrink-0"></span>
          <span>登录</span>
        </button>
      </div>
    );
  }

  return (
    <div className="border-t border-gray-800 pt-3">
      <div className="relative">
        {showMenu && (
          <div className="absolute bottom-full left-0 right-0 mb-1 bg-gray-800 border border-gray-700 rounded shadow-lg">
            <button
              onClick={goToSettings}
              className="w-full px-3 py-2 text-sm text-gray-300 hover:bg-gray-700 text-left rounded-t"
            >
              设置
            </button>
            <button
              onClick={handleLogout}
              className="w-full px-3 py-2 text-sm text-gray-300 hover:bg-gray-700 text-left rounded-b"
            >
              退出登录
            </button>
          </div>
        )}
        <button
          onClick={() => setShowMenu(!showMenu)}
          className="w-full px-3 py-2 rounded hover:bg-gray-800 text-gray-300 hover:text-white text-sm text-left flex items-center gap-2"
        >
          <span className="relative flex h-2 w-2 shrink-0">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
          </span>
          <span className="flex-1 truncate">{user.username}</span>
          <span className="text-xs text-gray-500">{showMenu ? "▲" : "▼"}</span>
        </button>
      </div>
    </div>
  );
}
