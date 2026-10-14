package security

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func NewPasetoVerifierFromEnv() (*PasetoVerifier, error) {
	pubKeyHex := strings.TrimSpace(os.Getenv("PASETO_PUBLIC_KEY_HEX"))
	if pubKeyHex == "" {
		return nil, fmt.Errorf("PASETO_PUBLIC_KEY_HEX environment variable is required")
	}

	rawBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hex encoding in PASETO_PUBLIC_KEY_HEX: %w", err)
	}

	if len(rawBytes) != ed25519.PublicKeySize { // 32 bytes
		return nil, fmt.Errorf("PASETO_PUBLIC_KEY_HEX must be exactly %d bytes (64 hex characters), got %d bytes", ed25519.PublicKeySize, len(rawBytes))
	}

	pubKey := ed25519.PublicKey(rawBytes)

	verifier, err := NewPasetoVerifier(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create paseto verifier: %w", err)
	}

	return verifier, nil
}
