package websocket

import (
	"context"
	"github.com/MohsenDowlati/wassup/internal/core/domain"
	ports "github.com/MohsenDowlati/wassup/internal/core/ports"
	"log/slog"
	"sync"
)

type Hub struct {
	mu          sync.RWMutex
	connections map[string]map[string]ports.LocalClient
	logger      *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		connections: make(map[string]map[string]ports.LocalClient),
		logger:      logger,
	}
}

func (h *Hub) Register(client ports.LocalClient) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	sessions, exists := h.connections[client.UserID()]
	if !exists {
		sessions = make(map[string]ports.LocalClient)
		h.connections[client.UserID()] = sessions
	}

	sessions[client.SessionID()] = client
	h.logger.Debug("session registered in hub", "user_id", client.UserID(), "session_id", client.SessionID())
	return nil
}

func (h *Hub) Unregister(userID, sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sessions, exists := h.connections[userID]
	if !exists {
		return
	}

	delete(sessions, sessionID)
	if len(sessions) == 0 {
		delete(h.connections, userID)
	}
	h.logger.Debug("session unregistered from hub", "user_id", userID, "session_id", sessionID)
}

func (h *Hub) RouteLocal(userID string, env *domain.Envelope) bool {
	h.mu.RLock()
	sessions, exists := h.connections[userID]
	if !exists || len(sessions) == 0 {
		h.mu.RUnlock()
		return false
	}

	targets := make([]ports.LocalClient, 0, len(sessions))
	for _, client := range sessions {
		targets = append(targets, client)
	}
	h.mu.RUnlock()

	dispatched := false
	for _, client := range targets {
		if client.Enqueue(env) {
			dispatched = true
		}
	}
	return dispatched
}

func (h *Hub) ActiveConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, sessions := range h.connections {
		total += len(sessions)
	}
	return total
}

func (h *Hub) BroadcastShutdown(ctx context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.logger.Info("draining local connections for node shutdown...")
	for _, sessions := range h.connections {
		for _, client := range sessions {
			client.Close(1001, "node draining for restart")
		}
	}
	h.connections = make(map[string]map[string]ports.LocalClient)
}
