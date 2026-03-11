"use client";

import { useEffect, useRef, useCallback, useState } from "react";
import type { ContainerInfo, ContainerStats } from "./api";

const WS_BASE = (process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080").replace(/^http/, "ws");

type WSMessage =
  | { type: "containers"; data: ContainerInfo[] }
  | { type: "container_stats"; containerId: string; data: ContainerStats };

type Listener = (msg: WSMessage) => void;

// Singleton WebSocket manager
let ws: WebSocket | null = null;
let listeners = new Set<Listener>();
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectDelay = 1000;
let manualClose = false;

function getToken(): string {
  if (typeof window === "undefined") return "";
  return localStorage.getItem("auth_token") || "";
}

function connect() {
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return;
  }

  const token = getToken();
  if (!token) return;

  manualClose = false;
  ws = new WebSocket(`${WS_BASE}/api/ws?token=${token}`);

  ws.onopen = () => {
    reconnectDelay = 1000;
  };

  ws.onmessage = (ev) => {
    try {
      const msg: WSMessage = JSON.parse(ev.data);
      listeners.forEach((fn) => fn(msg));
    } catch {
      // ignore malformed messages
    }
  };

  ws.onclose = (ev) => {
    ws = null;
    if (manualClose) return;

    // Auth failure — don't reconnect
    if (ev.code === 4401 || ev.code === 1008) {
      document.cookie = "auth_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT";
      window.location.href = "/login";
      return;
    }

    // Exponential backoff reconnect
    reconnectTimer = setTimeout(() => {
      reconnectDelay = Math.min(reconnectDelay * 2, 30000);
      connect();
    }, reconnectDelay);
  };

  ws.onerror = () => {
    // onclose will fire after this
  };
}

function disconnect() {
  manualClose = true;
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (ws) {
    ws.close();
    ws = null;
  }
}

function sendAction(action: string, containerId: string) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ action, containerId }));
  }
}

/**
 * Hook: subscribe to container list updates via WebSocket.
 * Falls back to HTTP polling if WS fails to connect.
 */
export function useContainerList() {
  const [containers, setContainers] = useState<ContainerInfo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let fallbackTimer: ReturnType<typeof setTimeout> | null = null;

    const handler: Listener = (msg) => {
      if (msg.type === "containers") {
        setContainers(msg.data);
        setLoading(false);
        // Cancel fallback if WS delivers data
        if (fallbackTimer) {
          clearTimeout(fallbackTimer);
          fallbackTimer = null;
        }
      }
    };

    listeners.add(handler);
    connect();

    // Fallback: if no WS data within 3s, fetch via HTTP
    fallbackTimer = setTimeout(async () => {
      if (loading) {
        try {
          const { api } = await import("./api");
          const data = await api.listContainers();
          setContainers(data);
        } catch {
          // ignore
        }
        setLoading(false);
      }
    }, 3000);

    return () => {
      listeners.delete(handler);
      if (fallbackTimer) clearTimeout(fallbackTimer);
      if (listeners.size === 0) {
        disconnect();
      }
    };
  }, []);

  return { containers, loading };
}

/**
 * Hook: subscribe to a specific container's stats via WebSocket.
 */
export function useContainerStats(containerId: string | null) {
  const [stats, setStats] = useState<ContainerStats | null>(null);
  const prevId = useRef<string | null>(null);

  useEffect(() => {
    if (!containerId) return;

    const handler: Listener = (msg) => {
      if (msg.type === "container_stats" && msg.containerId === containerId) {
        setStats(msg.data);
      }
    };

    listeners.add(handler);
    connect();

    // Subscribe
    sendAction("subscribe_stats", containerId);
    prevId.current = containerId;

    return () => {
      listeners.delete(handler);
      sendAction("unsubscribe_stats", containerId);
      if (listeners.size === 0) {
        disconnect();
      }
    };
  }, [containerId]);

  return stats;
}

/**
 * Force refresh: request the server to re-send container list.
 * (Useful after create/delete/start/stop operations)
 */
export function refreshContainers() {
  sendAction("refresh", "");
}
