package domain

import "errors"

var (
	ErrDuplicateMessage   = errors.New("domain: duplicate message detected (idempotent ignore)")
	ErrRecipientNotFound  = errors.New("domain: recipient session not found in active registry")
	ErrRateLimitExceeded  = errors.New("domain: client message rate limit exceeded")
	ErrConnectionDrained  = errors.New("domain: connection is currently draining")
	ErrInvalidCredentials = errors.New("domain: invalid authentication handshake token")
	ErrPayloadTooLarge    = errors.New("domain: payload size exceeds protocol constraints")
)
