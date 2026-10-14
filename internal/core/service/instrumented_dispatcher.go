package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/MohsenDowlati/wassup/internal/core/domain"
	"github.com/MohsenDowlati/wassup/internal/core/ports"
	"github.com/MohsenDowlati/wassup/internal/telemetry"
)

type InstrumentedDispatcher struct {
	registry    ports.SessionRegistry
	pubsub      ports.DistributedPubSub
	localHub    ports.LocalConnectionHub
	offlineRepo ports.OfflineMessageRepository
	nodeID      string
	offlineTTL  time.Duration
	metrics     *telemetry.Metrics
	logger      *slog.Logger
}

func NewInstrumentedDispatcher(
	registry ports.SessionRegistry,
	pubsub ports.DistributedPubSub,
	localHub ports.LocalConnectionHub,
	offlineRepo ports.OfflineMessageRepository,
	nodeID string,
	offlineTTL time.Duration,
	metrics *telemetry.Metrics,
	logger *slog.Logger,
) *InstrumentedDispatcher {
	return &InstrumentedDispatcher{
		registry:    registry,
		pubsub:      pubsub,
		localHub:    localHub,
		offlineRepo: offlineRepo,
		nodeID:      nodeID,
		offlineTTL:  offlineTTL,
		metrics:     metrics,
		logger:      logger,
	}
}

func (d *InstrumentedDispatcher) RouteInbound(ctx context.Context, env *domain.Envelope) error {
	startTime := time.Now()
	d.metrics.MessagesReceived.WithLabelValues(string(env.Type)).Inc()

	if env.TraceID == "" {
		if tid, err := telemetry.GenerateTraceContext(); err == nil {
			env.TraceID = tid
		}
	}

	ctx = context.WithValue(ctx, telemetry.CtxTraceID, env.TraceID)
	ctx = context.WithValue(ctx, telemetry.CtxUserID, env.SenderID)

	if d.localHub.IsConnected(env.RecipientID) {
		if d.localHub.RouteLocal(env.RecipientID, env) {
			duration := time.Since(startTime).Seconds()
			d.metrics.DeliveryDuration.WithLabelValues("local").Observe(duration)
			d.metrics.MessagesRouted.WithLabelValues("local", "success").Inc()
			return nil
		}
	}

	nodeID, found, err := d.registry.FindNode(ctx, env.RecipientID)
	if err != nil {
		d.logger.ErrorContext(ctx, "failed to query connection registry", "error", err, "recipient_id", env.RecipientID)
	}

	if found && nodeID != d.nodeID {
		err = d.pubsub.PublishToNode(ctx, nodeID, env)
		if err == nil {
			duration := time.Since(startTime).Seconds()
			d.metrics.DeliveryDuration.WithLabelValues("remote_redis").Observe(duration)
			d.metrics.MessagesRouted.WithLabelValues("remote_redis", "success").Inc()
			return nil
		}
		d.logger.WarnContext(ctx, "failed remote node publish, falling back to offline storage", "error", err, "target_node", nodeID)
	}

	offlineMsg := domain.NewOfflineMessage(env, d.offlineTTL)
	if err := d.offlineRepo.Save(ctx, offlineMsg); err != nil {
		d.metrics.MessagesRouted.WithLabelValues("offline_mongo", "failed").Inc()
		d.logger.ErrorContext(ctx, "failed to persist offline message", "error", err, "idempotency_id", env.IdempotencyID)
		return err
	}

	duration := time.Since(startTime).Seconds()
	d.metrics.DeliveryDuration.WithLabelValues("offline_mongo").Observe(duration)
	d.metrics.MessagesRouted.WithLabelValues("offline_mongo", "success").Inc()
	d.metrics.OfflineMessagesPersisted.Inc()

	return nil
}
