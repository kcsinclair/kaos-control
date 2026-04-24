// Package hub manages WebSocket broadcast to connected clients.
package hub

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// Event is a typed JSON message broadcast to all connected WebSocket clients.
type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// Hub fans out events to a set of registered client send channels.
type Hub struct {
	mu      sync.Mutex
	clients map[chan<- []byte]struct{}
}

// New creates a Hub.
func New() *Hub {
	return &Hub{clients: make(map[chan<- []byte]struct{})}
}

// Register adds ch as a recipient for future Broadcast calls.
func (h *Hub) Register(ch chan<- []byte) {
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
}

// Unregister removes ch; subsequent broadcasts will not reach it.
func (h *Hub) Unregister(ch chan<- []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

// Broadcast serialises evt and delivers it to every registered client.
// Clients whose channels are full receive a drop warning rather than blocking.
func (h *Hub) Broadcast(evt Event) {
	data, err := json.Marshal(evt)
	if err != nil {
		slog.Error("hub: marshal event", "err", err)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- data:
		default:
			slog.Warn("hub: client channel full, dropping event", "type", evt.Type)
		}
	}
}
