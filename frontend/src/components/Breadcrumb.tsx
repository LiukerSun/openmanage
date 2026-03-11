"use client";

import { useEffect, useState } from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";

export function Breadcrumb() {
  const pathname = usePathname();
  const [containerName, setContainerName] = useState<string>("");

  const segments = pathname.split("/").filter(Boolean);

  // Extract container ID if on a container page
  const containerIdx = segments.indexOf("containers");
  const containerId = containerIdx >= 0 ? segments[containerIdx + 1] : null;

  useEffect(() => {
    if (!containerId) {
      setContainerName("");
      return;
    }
    api.getContainer(containerId)
      .then((c) => setContainerName(c.name || containerId.slice(0, 12)))
      .catch(() => setContainerName(containerId.slice(0, 12)));
  }, [containerId]);

  const subPageLabels: Record<string, string> = {
    logs: "日志",
    files: "文件",
    conversations: "对话",
  };

  // Build breadcrumb items
  const items: { label: string; href?: string }[] = [];

  if (pathname === "/") {
    items.push({ label: "仪表盘" });
  } else {
    items.push({ label: "仪表盘", href: "/" });

    if (segments[0] === "containers" && containerId) {
    // Container detail or sub-page
    const subPage = segments[containerIdx + 2];
    if (subPage) {
      items.push({ label: containerName || "...", href: `/containers/${containerId}` });
      items.push({ label: subPageLabels[subPage] || subPage });
    } else {
      items.push({ label: containerName || "..." });
    }
  } else if (segments[0] === "create") {
    items.push({ label: "新建容器" });
  } else if (segments[0] === "templates") {
    items.push({ label: "模板管理" });
  } else if (segments[0] === "settings") {
    items.push({ label: "设置" });
  }
  }

  return (
    <nav className="flex items-center gap-1.5 text-sm text-gray-400 mb-4">
      {items.map((item, i) => (
        <span key={i} className="flex items-center gap-1.5">
          {i > 0 && <span className="text-gray-600">/</span>}
          {item.href ? (
            <Link href={item.href} className="hover:text-white transition-colors">
              {item.label}
            </Link>
          ) : (
            <span className="text-gray-200">{item.label}</span>
          )}
        </span>
      ))}
    </nav>
  );
}
