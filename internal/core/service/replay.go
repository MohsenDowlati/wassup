package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/MohsenDowlati/wassup/internal/core/domain"
	ports "github.com/MohsenDowlati/wassup/internal/core/ports"
)

const (
	replayBatchSize = 100
	replayLease     = 2 * time.Minute
)

type ReplayService struct {
	repository ports.OfflineMessageRepository
	localHub   ports.LocalConnectionHub
	logger     *slog.Logger
}

func NewReplayService(
	repository ports.OfflineMessageRepository,
	localHub ports.LocalConnectionHub,
	logger *slog.Logger,
) *ReplayService {
	return &ReplayService{
		repository: repository,
		localHub:   localHub,
		logger:     logger,
	}
}

func (s *ReplayService) Replay(
	ctx context.Context,
	recipientID string,
) error {
	items, err := s.repository.ClaimPending(
		ctx,
		recipientID,
		replayBatchSize,
		replayLease,
	)
	if err != nil {
		return err
	}

	for i, item := range items {
		env := &domain.Envelope{
			IdempotencyID:  item.IdempotencyID,
			ConversationID: item.ConversationID,
			SequenceID:     item.SequenceID,
			Type:           domain.TypeMessage,
			SenderID:       item.SenderID,
			RecipientID:    item.RecipientID,
			Payload: domain.CiphertextPayload{
				Ciphertext:  append([]byte(nil), item.Payload.Ciphertext...),
				CipherSuite: item.Payload.CipherSuite,
				KeyID:       item.Payload.KeyID,
				Nonce:       append([]byte(nil), item.Payload.Nonce...),
			},
			Timestamp: item.CreatedAt.UnixMilli(),
		}

		if !s.localHub.RouteLocal(recipientID, env) {
			s.logger.Debug("recipient disconnected during replay, releasing leases",
				"recipient_id", recipientID,
				"idempotency_id", item.IdempotencyID,
			)

			// Release lease on the failed item and all remaining un-attempted items
			// so they are immediately available if the user reconnects immediately.
			for _, remaining := range items[i:] {
				if relErr := s.repository.ReleaseLease(ctx, recipientID, remaining.IdempotencyID); relErr != nil {
					s.logger.Warn("failed to release lease on disconnect",
						"recipient_id", recipientID,
						"idempotency_id", remaining.IdempotencyID,
						"err", relErr,
					)
				}
			}
			return nil
		}
	}

	return nil
}
