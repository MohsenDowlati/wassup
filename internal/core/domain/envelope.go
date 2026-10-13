package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

type EnvelopeType string

const (
	TypeMessage EnvelopeType = "MESSAGE"

	TypeServerAck EnvelopeType = "SERVER_ACK"

	TypeClientAck EnvelopeType = "CLIENT_ACK"

	TypeHeartbeat EnvelopeType = "HEARTBEAT"

	TypeSyncRequest EnvelopeType = "SYNC_REQ"
)

type DeliveryStatus string

const (
	StatusPending   DeliveryStatus = "PENDING"
	StatusDelivered DeliveryStatus = "DELIVERED"
	StatusAcked     DeliveryStatus = "ACKNOWLEDGED"
	StatusFailed    DeliveryStatus = "FAILED"
)

type Envelope struct {
	IdempotencyID string `json:"idempotency_id" bson:"idempotency_id"`

	SequenceID uint64 `json:"seq_id,omitempty" bson:"seq_id,omitempty"`

	Type EnvelopeType `json:"type" bson:"type"`

	SenderID    string `json:"sender_id" bson:"sender_id"`
	RecipientID string `json:"recipient_id" bson:"recipient_id"`

	Payload []byte `json:"payload" bson:"payload"`

	Timestamp int64 `json:"timestamp" bson:"timestamp"`

	TraceID string `json:"trace_id,omitempty" bson:"trace_id,omitempty"`
}

func (e *Envelope) Validate() error {
	if e.IdempotencyID == "" {
		return errors.New("envelope: missing idempotency_id")
	}
	if e.Type == "" {
		return errors.New("envelope: missing type")
	}
	switch e.Type {
	case TypeMessage:
		if e.SenderID == "" || e.RecipientID == "" {
			return errors.New("envelope: message requires both sender_id and recipient_id")
		}
		if len(e.Payload) == 0 {
			return errors.New("envelope: empty payload")
		}
		if len(e.Payload) > 64*1024 { // 64 KiB strict frame limit
			return errors.New("envelope: payload exceeds maximum allowable size (64KiB)")
		}
	case TypeClientAck, TypeServerAck:
		if e.SenderID == "" || e.RecipientID == "" {
			return errors.New("envelope: ack requires sender_id and recipient_id")
		}
	}
	return nil
}

func NewServerAck(targetID, senderID, recipientID string) *Envelope {
	return &Envelope{
		IdempotencyID: generateSecureHex(16),
		Type:          TypeServerAck,
		SenderID:      senderID,
		RecipientID:   recipientID,
		Payload:       []byte(targetID),
		Timestamp:     time.Now().UTC().UnixMilli(),
	}
}

func generateSecureHex(byteLen int) string {
	b := make([]byte, byteLen)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
