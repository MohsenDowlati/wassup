package ports

import (
	"context"
	"github.com/MohsenDowlati/wassup/internal/core/domain"
)

type OfflineMessageRepository interface {
	StoreOffline(ctx context.Context, msg *domain.OfflineMessage) error

	FetchPending(ctx context.Context, recipientID string, limit int) ([]*domain.OfflineMessage, error)

	AcknowledgeAndPurge(ctx context.Context, idempotencyID string, recipientID string) error

	EnsureIndexes(ctx context.Context) error
}
