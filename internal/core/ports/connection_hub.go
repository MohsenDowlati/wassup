package ports

import (
	"context"
	"github.com/MohsenDowlati/wassup/internal/core/domain"
)

type InboundMessageRouter interface {
	RouteInbound(ctx context.Context, env *domain.Envelope) error
}

type LocalConnectionHub interface {
	Register(client LocalClient) error

	Unregister(userID, sessionID string)

	RouteLocal(userID string, env *domain.Envelope) bool

	ActiveConnections() int

	BroadcastShutdown(ctx context.Context)
	IsConnected(userID string) bool
}

type LocalClient interface {
	UserID() string
	SessionID() string
	Enqueue(env *domain.Envelope) bool
	Close(code int, reason string)
}
