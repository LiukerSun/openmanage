const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export interface LoginResponse {
  success: boolean;
  message?: string;
}

export interface AuthStatus {
  authenticated: boolean;
  username?: string;
}

export interface ContainerInfo {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  created: number;
  labels: Record<string, string>;
  mounts: { source: string; destination: string; rw: boolean }[];
}

export interface FileEntry {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
}

export interface FileContent {
  path: string;
  content: string;
}

export interface Conversation {
  sessionId: string;
  title: string;
  updatedAt: string;
}

export interface Message {
  role: string;
  content: string;
}

export interface ConversationDetail {
  sessionId: string;
  title: string;
  messages: Message[];
}

export interface CreateContainerRequest {
  name: string;
  image: string;
  dataPath: string;
  port: number;
  env: Record<string, string>;
  description?: string;
}

export interface ContainerStats {
  cpuPercent: number;
  memUsage: number;
  memLimit: number;
  memPercent: number;
  netRx: number;
  netTx: number;
  pids: number;
}

export interface UserPreferences {
  username: string;
  style: string;
  tools: string;
  extraContext: string;
  variables?: Record<string, string>;
  discourseUrl?: string;
  discourseApiKey?: string;
  discourseCategory?: string;
}

export interface BatchCreateRequest {
  prefix: string;
  count: number;
  startPort: number;
  image?: string;
  description?: string;
}

export interface ForumAction {
  type: string;
  title: string;
  topicId: number;
  postNumber: number;
  createdAt: string;
  excerpt: string;
  slug: string;
}

export interface ForumActivity {
  username: string;
  topicCount: number;
  postCount: number;
  likesGiven: number;
  likesReceived: number;
  daysVisited: number;
  actions: ForumAction[];
}

export interface BatchChatRequest {
  containerIds: string[];
  message: string;
}

export interface CronJob {
  id: string;
  name: string;
  schedule: string;
  prompt: string;
  enabled: boolean;
  lastRun?: string;
  nextRun?: string;
}

export interface HeartbeatConfig {
  every: string;
  mode?: string;
}

export interface CronListResponse {
  jobs: CronJob[];
  heartbeat: HeartbeatConfig | null;
}

async function fetchAPI<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    credentials: "include",
  });
  if (!res.ok) {
    if (res.status === 401 && typeof window !== "undefined" && !path.includes("/api/auth/")) {
      // Clear the expired token cookie so middleware won't block the login page
      document.cookie = "auth_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT";
      window.location.href = "/login";
    }
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export const api = {
  listContainers: () => fetchAPI<ContainerInfo[]>("/api/containers"),

  createContainer: (req: CreateContainerRequest) =>
    fetchAPI<{ id: string; status: string }>("/api/create-container", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req),
    }),

  getContainer: (id: string) => fetchAPI<ContainerInfo>(`/api/containers/${id}`),

  startContainer: (id: string) =>
    fetchAPI(`/api/containers/${id}/start`, { method: "POST" }),

  stopContainer: (id: string) =>
    fetchAPI(`/api/containers/${id}/stop`, { method: "POST" }),

  restartContainer: (id: string) =>
    fetchAPI(`/api/containers/${id}/restart`, { method: "POST" }),

  deleteContainer: (id: string) =>
    fetchAPI(`/api/containers/${id}`, { method: "DELETE" }),

  updateContainer: (id: string, data: { name?: string }) =>
    fetchAPI(`/api/containers/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    }),

  getContainerStats: (id: string) =>
    fetchAPI<ContainerStats>(`/api/containers/${id}/stats`),

  listFiles: (id: string) => fetchAPI<FileEntry[]>(`/api/containers/${id}/files`),

  readFile: (id: string, path: string) =>
    fetchAPI<FileContent>(`/api/containers/${id}/files/${path}`),

  readDir: (id: string, path: string) =>
    fetchAPI<FileEntry[]>(`/api/containers/${id}/files/${path}`),

  writeFile: (id: string, path: string, content: string) =>
    fetchAPI(`/api/containers/${id}/files/${path}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path, content }),
    }),

  listConversations: (id: string) =>
    fetchAPI<Conversation[]>(`/api/containers/${id}/conversations`),

  getConversation: (id: string, sid: string) =>
    fetchAPI<ConversationDetail>(`/api/containers/${id}/conversations/${sid}`),

  logsURL: (id: string, tail = "200", follow = true) =>
    `${API_BASE}/api/containers/${id}/logs?tail=${tail}&follow=${follow}`,

  // Templates
  listTemplates: () => fetchAPI<FileEntry[]>("/api/templates"),

  readTemplate: (name: string) =>
    fetchAPI<FileContent>(`/api/templates/${name}`),

  writeTemplate: (name: string, content: string) =>
    fetchAPI(`/api/templates/${name}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: name, content }),
    }),

  createTemplate: (name: string, content: string) =>
    fetchAPI(`/api/templates`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path: name, content }),
    }),

  deleteTemplate: (name: string) =>
    fetchAPI(`/api/templates/${name}`, { method: "DELETE" }),

  // Auth
  login: (username: string, password: string) =>
    fetchAPI<LoginResponse>("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    }),

  logout: () =>
    fetchAPI<{ success: boolean }>("/api/auth/logout", { method: "POST" }),

  checkAuth: () =>
    fetchAPI<AuthStatus>("/api/auth/check"),

  changePassword: (oldPassword: string, newPassword: string) =>
    fetchAPI<{ success: boolean; message?: string }>("/api/auth/change-password", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ oldPassword, newPassword }),
    }),

  // Preferences
  getPreferences: () => fetchAPI<UserPreferences>("/api/preferences"),

  savePreferences: (prefs: UserPreferences) =>
    fetchAPI<{ status: string }>("/api/preferences", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(prefs),
    }),

  // Forum activity
  getForumActivity: (id: string) =>
    fetchAPI<ForumActivity>(`/api/containers/${id}/forum-activity`),

  // Chat - returns SSE URL for streaming
  chatURL: (id: string) => `${API_BASE}/api/containers/${id}/chat`,
  batchChatURL: () => `${API_BASE}/api/batch/chat`,

  // Cron & Heartbeat
  listCronJobs: (id: string) =>
    fetchAPI<CronListResponse>(`/api/containers/${id}/cron`),

  addCronJob: (id: string, name: string, schedule: string, prompt: string) =>
    fetchAPI<{ id: string }>(`/api/containers/${id}/cron`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, schedule, prompt }),
    }),

  toggleCronJob: (id: string, jobId: string, enabled: boolean) =>
    fetchAPI(`/api/containers/${id}/cron/${jobId}/toggle`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    }),

  runCronJob: (id: string, jobId: string) =>
    fetchAPI(`/api/containers/${id}/cron/${jobId}/run`, { method: "POST" }),

  removeCronJob: (id: string, jobId: string) =>
    fetchAPI(`/api/containers/${id}/cron/${jobId}`, { method: "DELETE" }),

  updateHeartbeat: (id: string, every: string) =>
    fetchAPI(`/api/containers/${id}/heartbeat`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ every }),
    }),

  generateCronJob: (id: string, description: string) =>
    fetchAPI<{ schedule: string; prompt: string }>(`/api/containers/${id}/cron/generate`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ description }),
    }),
};
