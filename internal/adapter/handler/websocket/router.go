package websocket

import (
	"crypto/rand"
	"encoding/hex"
	ports "github.com/MohsenDowlati/wassup/internal/core/port"
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
)

type GatewayHandler struct {
	hub            ports.LocalConnectionHub
	router         ports.InboundMessageRouter
	logger         *slog.Logger
	allowedOrigins []string
}

func NewGatewayHandler(
	hub ports.LocalConnectionHub,
	router ports.InboundMessageRouter,
	logger *slog.Logger,
	allowedOrigins []string,
) *GatewayHandler {
	return &GatewayHandler{
		hub:            hub,
		router:         router,
		logger:         logger,
		allowedOrigins: allowedOrigins,
	}
}

func (gh *GatewayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Unauthorized: missing user_id query parameter", http.StatusUnauthorized)
		return
	}

	opts := &websocket.AcceptOptions{
		OriginPatterns: gh.allowedOrigins,
	}

	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		gh.logger.Warn("handshake failed", "err", err)
		return
	}

	sessionID := generateSessionID()
	session := NewClientSession(
		r.Context(),
		userID,
		sessionID,
		conn,
		gh.hub,
		gh.router,
		gh.logger,
	)

	if err := gh.hub.Register(session); err != nil {
		_ = conn.Close(websocket.StatusInternalError, "internal registry failure")
		return
	}

	session.Start()
}

func generateSessionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
