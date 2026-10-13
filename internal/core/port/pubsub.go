package ports

import (
	"context"
	"github.com/MohsenDowlati/wassup/internal/core/domain"
)

type MessageHandler func(ctx context.Context, env *domain.Envelope) error

type DistributedPubSub interface {
	PublishToGateway(ctx context.Context, targetGatewayID string, env *domain.Envelope) error

	SubscribeGatewayInbox(ctx context.Context, gatewayID string, handler MessageHandler) error

	Close() error
}

type SessionRegistry interface {
	RegisterSession(ctx context.Context, userID, gatewayID string) error

	UnregisterSession(ctx context.Context, userID, gatewayID string) error

	LocateUser(ctx context.Context, userID string) (string, error)

	RefreshSession(ctx context.Context, userID, gatewayID string) error
}
