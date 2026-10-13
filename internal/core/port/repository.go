package ports

import (
	"context"
	"github.com/MohsenDowlati/wassup/internal/core/domain"
	"time"
)

type OfflineMessageRepository interface {
	StoreOffline(ctx context.Context, msg *domain.OfflineMessage) error

	ClaimPending(
		ctx context.Context,
		recipientID string,
		limit int,
		lease time.Duration,
	) ([]*domain.OfflineMessage, error)

	AcknowledgeAndPurge(
		ctx context.Context,
		recipientID string,
		idempotencyID string,
	) error
	
	ReleaseLease(
		ctx context.Context,
		recipientID string,
		idempotencyID string,
	) error

	EnsureIndexes(ctx context.Context) error
}
