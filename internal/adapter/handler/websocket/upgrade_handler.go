package websocket

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/MohsenDowlati/wassup/internal/core/ports"
	"github.com/MohsenDowlati/wassup/internal/security"
	"github.com/coder/websocket"
)

type SecureUpgradeHandler struct {
	verifier        ports.TokenVerifier
	rateLimiter     ports.DistributedRateLimiter
	originValidator *security.OriginValidator
	hub             ports.LocalConnectionHub
	router          ports.InboundMessageRouter
	logger          *slog.Logger

	ipRateLimitWindow   time.Duration
	ipRateLimitQuota    int64
	userConnLimitWindow time.Duration
	userConnLimitQuota  int64
}

func NewSecureUpgradeHandler(
	verifier ports.TokenVerifier,
	rateLimiter ports.DistributedRateLimiter,
	originValidator *security.OriginValidator,
	hub ports.LocalConnectionHub,
	router ports.InboundMessageRouter,
	logger *slog.Logger,
) *SecureUpgradeHandler {
	return &SecureUpgradeHandler{
		verifier:            verifier,
		rateLimiter:         rateLimiter,
		originValidator:     originValidator,
		hub:                 hub,
		router:              router,
		logger:              logger,
		ipRateLimitWindow:   1 * time.Minute,
		ipRateLimitQuota:    60,
		userConnLimitWindow: 10 * time.Second,
		userConnLimitQuota:  5,
	}
}

func (h *SecureUpgradeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !h.originValidator.IsOriginAllowed(r) {
		h.logger.Warn("origin rejection", "origin", r.Header.Get("Origin"), "remote", r.RemoteAddr)
		http.Error(w, "Forbidden Origin", http.StatusForbidden)
		return
	}

	ip := extractClientIP(r)
	allowed, _, retryAfter, err := h.rateLimiter.Allow(ctx, "ip:"+ip, h.ipRateLimitQuota, h.ipRateLimitWindow)
	if err != nil {
		h.logger.Error("rate limiter failure (ip)", "error", err, "ip", ip)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	token, err := extractToken(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	claims, err := h.verifier.Verify(token)
	if err != nil {
		h.logger.Warn("authentication failure", "error", err, "remote", r.RemoteAddr)
		http.Error(w, "Unauthorized: Invalid or expired token", http.StatusUnauthorized)
		return
	}

	allowed, _, retryAfter, err = h.rateLimiter.Allow(ctx, "user:"+claims.UserID, h.userConnLimitQuota, h.userConnLimitWindow)
	if err != nil {
		h.logger.Error("rate limiter failure (user)", "error", err, "user_id", claims.UserID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
		http.Error(w, "Too Many Connection Attempts", http.StatusTooManyRequests)
		return
	}

	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		CompressionMode:    websocket.CompressionDisabled,
	})
	if err != nil {
		h.logger.Error("websocket accept failure", "error", err, "user_id", claims.UserID)
		return
	}

	sessionID := GenerateSessionID()

	client := NewClientSession(
		// The HTTP request context is canceled when ServeHTTP returns after the
		// upgrade. The WebSocket session outlives that handler, so detach its
		// lifetime from the request; gateway shutdown closes sessions via the hub.
		context.WithoutCancel(r.Context()),
		claims.UserID,
		sessionID,
		wsConn,
		h.hub,
		h.router,
		h.logger,
	)

	if err := h.hub.Register(client); err != nil {
		h.logger.Error("hub registration failed", "error", err, "user_id", claims.UserID, "session_id", sessionID)
		_ = wsConn.Close(websocket.StatusInternalError, "registration failed")
		return
	}

	client.Start()
}

func extractToken(r *http.Request) (string, error) {
	if auth := r.Header.Get("Authorization"); auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return strings.TrimSpace(parts[1]), nil
		}
	}
	if token := r.URL.Query().Get("token"); token != "" {
		return strings.TrimSpace(token), nil
	}
	return "", errors.New("missing authorization token")
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
