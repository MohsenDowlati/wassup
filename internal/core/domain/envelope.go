package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

type EnvelopeType string

const (
	TypeMessage     EnvelopeType = "MESSAGE"
	TypeServerAck   EnvelopeType = "SERVER_ACK"
	TypeClientAck   EnvelopeType = "CLIENT_ACK"
	TypeHeartbeat   EnvelopeType = "HEARTBEAT"
	TypeSyncRequest EnvelopeType = "SYNC_REQ"
)

type DeliveryStatus string

const (
	StatusPending   DeliveryStatus = "PENDING"
	StatusDelivered DeliveryStatus = "DELIVERED"
	StatusAcked     DeliveryStatus = "ACKNOWLEDGED"
	StatusFailed    DeliveryStatus = "FAILED"
)

type CiphertextPayload struct {
	Ciphertext  []byte `json:"ciphertext" bson:"ciphertext"`
	CipherSuite string `json:"cipher_suite,omitempty" bson:"cipher_suite,omitempty"`
	KeyID       string `json:"key_id,omitempty" bson:"key_id,omitempty"`
	Nonce       []byte `json:"nonce,omitempty" bson:"nonce,omitempty"`
}

type Envelope struct {
	IdempotencyID  string            `json:"idempotency_id" bson:"idempotency_id"`
	SequenceID     uint64            `json:"seq_id,omitempty" bson:"seq_id,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty" bson:"conversation_id,omitempty"`
	Type           EnvelopeType      `json:"type" bson:"type"`
	SenderID       string            `json:"sender_id" bson:"sender_id"`
	RecipientID    string            `json:"recipient_id" bson:"recipient_id"`
	Payload        CiphertextPayload `json:"payload" bson:"payload"`
	Timestamp      int64             `json:"timestamp" bson:"timestamp"`
	TraceID        string            `json:"trace_id,omitempty" bson:"trace_id,omitempty"`
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
		if len(e.Payload.Ciphertext) == 0 {
			return errors.New("envelope: empty message payload")
		}
		// Strict 64 KiB payload limit to prevent Gateway memory exhaustion
		if len(e.Payload.Ciphertext) > 64*1024 {
			return errors.New("envelope: payload exceeds maximum allowable size (64KiB)")
		}

	case TypeClientAck, TypeServerAck:
		if e.SenderID == "" || e.RecipientID == "" {
			return errors.New("envelope: ack requires sender_id and recipient_id")
		}
		if len(e.Payload.Ciphertext) == 0 {
			return errors.New("envelope: ack requires target idempotency_id in payload")
		}

	case TypeHeartbeat, TypeSyncRequest:
	}

	return nil
}

func NewServerAck(targetIdempotencyID, gatewayID, recipientID string) (*Envelope, error) {
	ackID, err := generateSecureHex(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate secure ack id: %w", err)
	}

	return &Envelope{
		IdempotencyID: ackID,
		Type:          TypeServerAck,
		SenderID:      gatewayID,
		RecipientID:   recipientID,
		Payload: CiphertextPayload{
			Ciphertext: []byte(targetIdempotencyID),
		},
		Timestamp: time.Now().UTC().UnixMilli(),
	}, nil
}

func generateSecureHex(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
