"use client";

import { ReactNode, useState } from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";
import { UserMenu } from "./UserMenu";
import { Breadcrumb } from "./Breadcrumb";

const navItems = [
  { href: "/", label: "仪表盘" },
  { href: "/create", label: "+ 新建容器" },
  { href: "/templates", label: "模板管理" },
];

function NavLink({ href, label, active }: { href: string; label: string; active: boolean }) {
  return (
    <Link
      href={href}
      className={`px-3 py-2 rounded text-sm transition-colors ${
        active
          ? "bg-blue-700 text-white"
          : "text-gray-300 hover:bg-gray-800 hover:text-white"
      }`}
    >
      {label}
    </Link>
  );
}

function AppContent({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  if (pathname === "/login") {
    return <>{children}</>;
  }

  const isActive = (href: string) => {
    if (href === "/") return pathname === "/";
    return pathname.startsWith(href);
  };

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Mobile header */}
      <div className="fixed top-0 left-0 right-0 z-40 bg-gray-900 border-b border-gray-800 p-3 flex items-center justify-between md:hidden">
        <Link href="/" className="text-lg font-bold text-white">OpenManage</Link>
        <button
          onClick={() => setSidebarOpen(!sidebarOpen)}
          className="p-2 rounded hover:bg-gray-800 text-gray-300"
          aria-label="菜单"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            {sidebarOpen ? (
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            ) : (
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
            )}
          </svg>
        </button>
      </div>

      {/* Overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <nav
        className={`fixed top-0 left-0 z-50 h-full w-56 bg-gray-900 border-r border-gray-800 p-4 flex flex-col transition-transform duration-200 md:translate-x-0 md:static md:z-auto ${
          sidebarOpen ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <Link href="/" className="text-lg font-bold text-white mb-4 block">
          OpenManage
        </Link>
        <div className="flex flex-col gap-1 flex-1">
          {navItems.map((item) => (
            <NavLink key={item.href} href={item.href} label={item.label} active={isActive(item.href)} />
          ))}
        </div>
        <UserMenu />
      </nav>

      {/* Main content */}
      <main className="flex-1 overflow-auto p-6 pt-16 md:pt-6">
        <Breadcrumb />
        {children}
      </main>
    </div>
  );
}

export function AppLayout({ children }: { children: ReactNode }) {
  return <AppContent>{children}</AppContent>;
}
