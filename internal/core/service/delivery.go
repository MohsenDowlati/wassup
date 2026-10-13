package services

import (
	"context"
	"fmt"
	"github.com/MohsenDowlati/wassup/internal/core/domain"
	ports "github.com/MohsenDowlati/wassup/internal/core/ports"
	"log/slog"
	"time"
)

type DeliveryService struct {
	repository ports.OfflineMessageRepository
	registry   ports.SessionRegistry
	pubsub     ports.DistributedPubSub
	localHub   ports.LocalConnectionHub
	gatewayID  string
	retention  time.Duration
	logger     *slog.Logger
}

func NewDeliveryService(
	repository ports.OfflineMessageRepository,
	registry ports.SessionRegistry,
	pubsub ports.DistributedPubSub,
	localHub ports.LocalConnectionHub,
	gatewayID string,
	retention time.Duration,
	logger *slog.Logger,
) *DeliveryService {
	return &DeliveryService{
		repository: repository,
		registry:   registry,
		pubsub:     pubsub,
		localHub:   localHub,
		gatewayID:  gatewayID,
		retention:  retention,
		logger:     logger,
	}
}

func (s *DeliveryService) AcceptMessage(
	ctx context.Context,
	env *domain.Envelope,
) error {
	if err := env.Validate(); err != nil {
		return fmt.Errorf("accept message: %w", err)
	}

	if s.localHub.RouteLocal(env.RecipientID, env) {
		s.logger.Debug("message queued to local recipient",
			"recipient_id", env.RecipientID,
			"idempotency_id", env.IdempotencyID,
		)
		return nil
	}

	targetGateway, err := s.registry.LocateUser(ctx, env.RecipientID)
	if err != nil {
		return fmt.Errorf("locate recipient gateway: %w", err)
	}

	if targetGateway == "" {
		return s.persistOffline(ctx, env)
	}

	if err := s.pubsub.PublishToGateway(ctx, targetGateway, env); err != nil {
		s.logger.Warn("gateway publish failed; persisting offline message",
			"target_gateway", targetGateway,
			"recipient_id", env.RecipientID,
			"err", err,
		)
		return s.persistOffline(ctx, env)
	}

	return nil
}

func (s *DeliveryService) persistOffline(
	ctx context.Context,
	env *domain.Envelope,
) error {
	msg := domain.NewOfflineMessage(env, s.retention)

	if err := s.repository.StoreOffline(ctx, msg); err != nil {
		return fmt.Errorf("persist offline message: %w", err)
	}

	return nil
}
