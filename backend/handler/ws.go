package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"openmanage/backend/docker"
	"openmanage/backend/model"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSMessage is the server→client message format.
type WSMessage struct {
	Type        string      `json:"type"`
	ContainerID string      `json:"containerId,omitempty"`
	Data        interface{} `json:"data"`
}

// WSClientMessage is the client→server message format.
type WSClientMessage struct {
	Action      string `json:"action"`
	ContainerID string `json:"containerId"`
}

// wsConn wraps a single WebSocket connection and its stats subscriptions.
type wsConn struct {
	conn       *websocket.Conn
	mu         sync.Mutex
	statsSubs  map[string]bool // container IDs subscribed for stats
	subsMu     sync.RWMutex
}

func (c *wsConn) send(msg WSMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	c.conn.WriteJSON(msg)
}

// WSHandler manages WebSocket connections.
type WSHandler struct {
	Docker    *docker.Client
	JWTSecret []byte

	mu    sync.RWMutex
	conns map[*wsConn]bool

	// cached container list for diff detection
	lastJSON string
}

func NewWSHandler(dockerClient *docker.Client, jwtSecret []byte) *WSHandler {
	h := &WSHandler{
		Docker:    dockerClient,
		JWTSecret: jwtSecret,
		conns:     make(map[*wsConn]bool),
	}
	go h.broadcastLoop()
	return h
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Authenticate via query param (WebSocket can't send custom headers)
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		// Fallback: try cookie
		if c, err := r.Cookie("token"); err == nil {
			tokenStr = c.Value
		}
	}
	if tokenStr == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.JWTSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	wc := &wsConn{
		conn:      conn,
		statsSubs: make(map[string]bool),
	}

	h.mu.Lock()
	h.conns[wc] = true
	h.mu.Unlock()

	// Send initial container list immediately
	h.sendContainerList(wc)

	// Read loop: handle subscribe/unsubscribe messages
	go h.readLoop(wc)
}

func (h *WSHandler) readLoop(wc *wsConn) {
	defer func() {
		h.mu.Lock()
		delete(h.conns, wc)
		h.mu.Unlock()
		wc.conn.Close()
	}()

	wc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wc.conn.SetPongHandler(func(string) error {
		wc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, msg, err := wc.conn.ReadMessage()
		if err != nil {
			break
		}

		var cm WSClientMessage
		if err := json.Unmarshal(msg, &cm); err != nil {
			continue
		}

		wc.subsMu.Lock()
		switch cm.Action {
		case "subscribe_stats":
			if cm.ContainerID != "" {
				wc.statsSubs[cm.ContainerID] = true
			}
		case "unsubscribe_stats":
			delete(wc.statsSubs, cm.ContainerID)
		}
		wc.subsMu.Unlock()
	}
}

func (h *WSHandler) sendContainerList(wc *wsConn) {
	containers, err := h.getContainerList()
	if err != nil {
		return
	}
	wc.send(WSMessage{Type: "containers", Data: containers})
}

func (h *WSHandler) getContainerList() ([]model.ContainerInfo, error) {
	containers, err := h.Docker.ListOpenClawContainers(context.Background())
	if err != nil {
		return nil, err
	}

	result := make([]model.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		mounts := make([]model.MountInfo, 0, len(c.Mounts))
		for _, m := range c.Mounts {
			mounts = append(mounts, model.MountInfo{
				Source:      m.Source,
				Destination: m.Destination,
				RW:          m.RW,
			})
		}
		result = append(result, model.ContainerInfo{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			State:   c.State,
			Status:  c.Status,
			Created: c.Created,
			Labels:  c.Labels,
			Mounts:  mounts,
		})
	}
	return result, nil
}

// broadcastLoop periodically checks container list and stats, pushes to clients.
func (h *WSHandler) broadcastLoop() {
	containerTicker := time.NewTicker(3 * time.Second)
	statsTicker := time.NewTicker(3 * time.Second)
	pingTicker := time.NewTicker(30 * time.Second)
	defer containerTicker.Stop()
	defer statsTicker.Stop()
	defer pingTicker.Stop()

	for {
		select {
		case <-containerTicker.C:
			h.broadcastContainers()

		case <-statsTicker.C:
			h.broadcastStats()

		case <-pingTicker.C:
			h.pingAll()
		}
	}
}

func (h *WSHandler) broadcastContainers() {
	containers, err := h.getContainerList()
	if err != nil {
		return
	}

	data, _ := json.Marshal(containers)
	current := string(data)

	// Only broadcast if changed
	if current == h.lastJSON {
		return
	}
	h.lastJSON = current

	msg := WSMessage{Type: "containers", Data: containers}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for wc := range h.conns {
		wc.send(msg)
	}
}

func (h *WSHandler) broadcastStats() {
	// Collect all subscribed container IDs
	needed := make(map[string]bool)
	h.mu.RLock()
	for wc := range h.conns {
		wc.subsMu.RLock()
		for id := range wc.statsSubs {
			needed[id] = true
		}
		wc.subsMu.RUnlock()
	}
	h.mu.RUnlock()

	if len(needed) == 0 {
		return
	}

	// Fetch stats for each subscribed container
	statsMap := make(map[string]*model.ContainerStats)
	for id := range needed {
		s, err := h.Docker.ContainerStats(context.Background(), id)
		if err != nil {
			continue
		}
		statsMap[id] = s
	}

	// Send to each connection only the stats they subscribed to
	h.mu.RLock()
	defer h.mu.RUnlock()
	for wc := range h.conns {
		wc.subsMu.RLock()
		for id := range wc.statsSubs {
			if s, ok := statsMap[id]; ok {
				wc.send(WSMessage{
					Type:        "container_stats",
					ContainerID: id,
					Data:        s,
				})
			}
		}
		wc.subsMu.RUnlock()
	}
}

func (h *WSHandler) pingAll() {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for wc := range h.conns {
		wc.mu.Lock()
		wc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		wc.conn.WriteMessage(websocket.PingMessage, nil)
		wc.mu.Unlock()
	}
}
