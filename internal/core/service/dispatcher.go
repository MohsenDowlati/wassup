package services

import (
	"context"
	"github.com/MohsenDowlati/wassup/internal/core/domain"
	ports "github.com/MohsenDowlati/wassup/internal/core/ports"
	"log/slog"
)

type Dispatcher struct {
	hub    ports.LocalConnectionHub
	logger *slog.Logger
}

func NewDispatcher(hub ports.LocalConnectionHub, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		hub:    hub,
		logger: logger,
	}
}

func (d *Dispatcher) HandleDistributedMessage(ctx context.Context, env *domain.Envelope) error {
	delivered := d.hub.RouteLocal(env.RecipientID, env)
	if !delivered {
		d.logger.Warn("user no longer present on local hub, delivery dropped",
			"recipient_id", env.RecipientID,
			"idempotency_id", env.IdempotencyID,
		)
		return domain.ErrRecipientNotFound
	}

	d.logger.Debug("dispatched envelope to local socket",
		"recipient_id", env.RecipientID,
		"idempotency_id", env.IdempotencyID,
	)
	return nil
}
