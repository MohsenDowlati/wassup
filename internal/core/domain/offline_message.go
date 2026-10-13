package domain

import "time"

type OfflineMessage struct {
	ID            string         `bson:"_id,omitempty" json:"id"`
	IdempotencyID string         `bson:"idempotency_id" json:"idempotency_id"`
	SenderID      string         `bson:"sender_id" json:"sender_id"`
	RecipientID   string         `bson:"recipient_id" json:"recipient_id"`
	Payload       []byte         `bson:"payload" json:"payload"`
	Status        DeliveryStatus `bson:"status" json:"status"`
	CreatedAt     time.Time      `bson:"created_at" json:"created_at"`

	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"`
}

func NewOfflineMessage(env *Envelope, ttl time.Duration) *OfflineMessage {
	now := time.Now().UTC()
	return &OfflineMessage{
		IdempotencyID: env.IdempotencyID,
		SenderID:      env.SenderID,
		RecipientID:   env.RecipientID,
		Payload:       env.Payload,
		Status:        StatusPending,
		CreatedAt:     now,
		ExpiresAt:     now.Add(ttl),
	}
}
