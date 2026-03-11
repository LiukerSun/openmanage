package model

type ContainerInfo struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Created int64             `json:"created"`
	Labels  map[string]string `json:"labels"`
	Mounts  []MountInfo       `json:"mounts"`
}

type MountInfo struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	RW          bool   `json:"rw"`
}

type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
}

type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type Conversation struct {
	SessionID string `json:"sessionId"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updatedAt"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ConversationDetail struct {
	SessionID string    `json:"sessionId"`
	Title     string    `json:"title"`
	Messages  []Message `json:"messages"`
}

type CreateContainerRequest struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	DataPath    string            `json:"dataPath"`
	Port        int               `json:"port"`
	Env         map[string]string `json:"env"`
	Description string            `json:"description"`
}

type UpdateContainerRequest struct {
	Name string `json:"name"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type ContainerStats struct {
	CPUPercent float64 `json:"cpuPercent"`
	MemUsage   uint64  `json:"memUsage"`
	MemLimit   uint64  `json:"memLimit"`
	MemPercent float64 `json:"memPercent"`
	NetRx      uint64  `json:"netRx"`
	NetTx      uint64  `json:"netTx"`
	PIDs       uint64  `json:"pids"`
}
