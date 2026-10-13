package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MohsenDowlati/wassup/internal/core/domain"
	ports "github.com/MohsenDowlati/wassup/internal/core/port"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const gatewayInboxChannel = "gw:%s:inbox"

type PubSubManager struct {
	client *redis.Client
	logger *slog.Logger

	mu     sync.Mutex
	pubsub *redis.PubSub
	closed bool
}

func NewPubSubManager(client *redis.Client, logger *slog.Logger) ports.DistributedPubSub {
	return &PubSubManager{
		client: client,
		logger: logger,
	}
}

func (p *PubSubManager) PublishToGateway(ctx context.Context, targetGatewayID string, env *domain.Envelope) error {
	channel := fmt.Sprintf(gatewayInboxChannel, targetGatewayID)

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("redis pubsub: failed to marshal envelope: %w", err)
	}

	err = p.client.Publish(ctx, channel, data).Err()
	if err != nil {
		return fmt.Errorf("redis pubsub: publish to %s failed: %w", channel, err)
	}

	p.logger.Debug("envelope routed to gateway inbox",
		"target_gw", targetGatewayID,
		"recipient_id", env.RecipientID,
		"idempotency_id", env.IdempotencyID,
	)
	return nil
}

func (p *PubSubManager) SubscribeGatewayInbox(ctx context.Context, gatewayID string, handler ports.MessageHandler) error {
	channel := fmt.Sprintf(gatewayInboxChannel, gatewayID)

	p.mu.Lock()
	p.pubsub = p.client.Subscribe(ctx, channel)
	p.mu.Unlock()

	_, err := p.pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("redis pubsub: failed to subscribe to channel %s: %w", channel, err)
	}

	p.logger.Info("subscribed to gateway inbox", "channel", channel)

	go func() {
		ch := p.pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				p.logger.Info("inbox listener shutting down", "channel", channel)
				return

			case msg, ok := <-ch:
				if !ok {
					p.logger.Warn("redis pubsub channel closed")
					return
				}

				var env domain.Envelope
				if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
					p.logger.Error("failed to decode envelope from pubsub", "err", err)
					continue
				}

				go func(targetEnv domain.Envelope) {
					handlerCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()

					if err := handler(handlerCtx, &targetEnv); err != nil {
						p.logger.Error("handler error processing inbound envelope",
							"err", err,
							"idempotency_id", targetEnv.IdempotencyID,
							"recipient_id", targetEnv.RecipientID,
						)
					}
				}(env)
			}
		}
	}()

	return nil
}

func (p *PubSubManager) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed || p.pubsub == nil {
		return nil
	}
	p.closed = true
	return p.pubsub.Close()
}
