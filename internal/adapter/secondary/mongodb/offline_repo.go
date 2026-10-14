package mongodb

import (
	"context"
	"errors"
	"fmt"
	"github.com/MohsenDowlati/wassup/internal/core/domain"
	"github.com/MohsenDowlati/wassup/internal/core/ports"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OfflineRepository struct {
	collection *mongo.Collection
}

func (r *OfflineRepository) Save(ctx context.Context, msg *domain.OfflineMessage) error {
	_, err := r.collection.InsertOne(ctx, msg)
	return err
}

func (r *OfflineRepository) ClaimPending(
	ctx context.Context,
	recipientID string,
	limit int,
	lease time.Duration,
) ([]*domain.OfflineMessage, error) {
	if limit <= 0 {
		return nil, errors.New("offline repository: limit must be positive")
	}

	now := time.Now().UTC()
	leaseUntil := now.Add(lease)

	filter := bson.M{
		"recipient_id": recipientID,
		"expires_at":   bson.M{"$gt": now},
		"$or": []bson.M{
			{"status": domain.StatusPending},
			{
				"status":      domain.StatusDelivered,
				"lease_until": bson.M{"$lte": now},
			},
			{
				"status":      domain.StatusDelivered,
				"lease_until": bson.M{"$exists": false},
			},
		},
	}

	sort := bson.D{
		{Key: "sequence_id", Value: 1},
		{Key: "created_at", Value: 1},
	}

	var claimed []*domain.OfflineMessage

	for len(claimed) < limit {
		update := bson.M{
			"$set": bson.M{
				"status":       domain.StatusDelivered,
				"lease_until":  leaseUntil,
				"last_attempt": now,
			},
			"$inc": bson.M{
				"attempt_count": 1,
			},
		}

		var item domain.OfflineMessage
		err := r.collection.FindOneAndUpdate(
			ctx,
			filter,
			update,
			options.FindOneAndUpdate().
				SetSort(sort).
				SetReturnDocument(options.After),
		).Decode(&item)

		if errors.Is(err, mongo.ErrNoDocuments) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("claim pending messages: %w", err)
		}

		claimed = append(claimed, &item)
	}

	return claimed, nil
}

func (r *OfflineRepository) AcknowledgeAndPurge(
	ctx context.Context,
	recipientID string,
	idempotencyID string,
) error {
	filter := bson.M{
		"recipient_id":   recipientID,
		"idempotency_id": idempotencyID,
	}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("purge acknowledged message: %w", err)
	}

	if result.DeletedCount == 0 {
		return nil
	}

	return nil
}

func (r *OfflineRepository) ReleaseLease(
	ctx context.Context,
	recipientID string,
	idempotencyID string,
) error {
	filter := bson.M{
		"recipient_id":   recipientID,
		"idempotency_id": idempotencyID,
	}

	update := bson.M{
		"$set": bson.M{
			"status": domain.StatusPending,
		},
		"$unset": bson.M{
			"lease_until":  "",
			"last_attempt": "",
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("release message lease: %w", err)
	}

	return nil
}

func NewOfflineRepository(
	db *mongo.Database,
	collectionName string,
) ports.OfflineMessageRepository {
	return &OfflineRepository{
		collection: db.Collection(collectionName),
	}
}

func (r *OfflineRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "recipient_id", Value: 1},
				{Key: "status", Value: 1},
				{Key: "sequence_id", Value: 1},
			},
			Options: options.Index().
				SetName("pending_replay_order"),
		},
		{
			Keys: bson.D{
				{Key: "recipient_id", Value: 1},
				{Key: "idempotency_id", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetName("recipient_idempotency_unique"),
		},
		{
			Keys: bson.D{
				{Key: "expires_at", Value: 1},
			},
			Options: options.Index().
				SetExpireAfterSeconds(0).
				SetName("offline_message_ttl"),
		},
	})
	if err != nil {
		return fmt.Errorf("create offline-message indexes: %w", err)
	}

	return nil
}

func (r *OfflineRepository) StoreOffline(
	ctx context.Context,
	msg *domain.OfflineMessage,
) error {
	if msg == nil {
		return errors.New("offline repository: nil message")
	}

	_, err := r.collection.InsertOne(ctx, msg)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("store offline message: %w", err)
	}

	return nil
}
