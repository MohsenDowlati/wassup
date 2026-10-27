package domain

import "time"

type OfflineMessage struct {
	ID             string            `bson:"_id,omitempty" json:"id"`
	IdempotencyID  string            `bson:"idempotency_id" json:"idempotency_id"`
	ConversationID string            `bson:"conversation_id,omitempty" json:"conversation_id,omitempty"`
	SequenceID     uint64            `bson:"sequence_id,omitempty" json:"sequence_id,omitempty"`
	SenderID       string            `bson:"sender_id" json:"sender_id"`
	RecipientID    string            `bson:"recipient_id" json:"recipient_id"`
	Payload        CiphertextPayload `bson:"payload" json:"payload"`
	Status         DeliveryStatus    `bson:"status" json:"status"`

	CreatedAt    time.Time  `bson:"created_at" json:"created_at"`
	ExpiresAt    time.Time  `bson:"expires_at" json:"expires_at"`
	LastAttempt  *time.Time `bson:"last_attempt,omitempty" json:"last_attempt,omitempty"`
	LeaseUntil   *time.Time `bson:"lease_until,omitempty" json:"lease_until,omitempty"`
	AttemptCount int32      `bson:"attempt_count" json:"attempt_count"`
}

func NewOfflineMessage(env *Envelope, ttl time.Duration) *OfflineMessage {
	now := time.Now().UTC()

	return &OfflineMessage{
		IdempotencyID:  env.IdempotencyID,
		ConversationID: env.ConversationID,
		SequenceID:     env.SequenceID,
		SenderID:       env.SenderID,
		RecipientID:    env.RecipientID,
		Payload: CiphertextPayload{
			Ciphertext:  append([]byte(nil), env.Payload.Ciphertext...),
			CipherSuite: env.Payload.CipherSuite,
			KeyID:       env.Payload.KeyID,
			Nonce:       append([]byte(nil), env.Payload.Nonce...),
		},
		Status:    StatusPending,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
}
