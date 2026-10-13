package security

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MohsenDowlati/wassup/internal/core/ports"
)

const pasetoV4PublicPrefix = "v4.public."

type PasetoVerifier struct {
	publicKey ed25519.PublicKey
}

func NewPasetoVerifier(pubKey ed25519.PublicKey) (*PasetoVerifier, error) {
	if len(pubKey) != ed25519.PublicKeySize {
		return nil, errors.New("paseto: invalid Ed25519 public key size")
	}
	return &PasetoVerifier{publicKey: pubKey}, nil
}

func (v *PasetoVerifier) Verify(token string) (*ports.UserIdentity, error) {
	if !strings.HasPrefix(token, pasetoV4PublicPrefix) {
		return nil, ports.ErrTokenMalformed
	}

	payloadAndSig := strings.TrimPrefix(token, pasetoV4PublicPrefix)
	raw, err := base64.RawURLEncoding.DecodeString(payloadAndSig)
	if err != nil {
		return nil, ports.ErrTokenMalformed
	}

	if len(raw) < ed25519.SignatureSize {
		return nil, ports.ErrTokenMalformed
	}

	body := raw[:len(raw)-ed25519.SignatureSize]
	sig := raw[len(raw)-ed25519.SignatureSize:]

	paeMsg := preAuthEncode([]byte("v4.public"), body, nil)

	if !ed25519.Verify(v.publicKey, paeMsg, sig) {
		return nil, ports.ErrTokenInvalidSig
	}

	var claims struct {
		Subject   string `json:"sub"`
		DeviceID  string `json:"dev_id,omitempty"`
		IssuedAt  int64  `json:"iat"`
		ExpiresAt int64  `json:"exp"`
	}

	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, fmt.Errorf("%w: %v", ports.ErrTokenMalformed, err)
	}

	now := time.Now().UTC().Unix()
	if claims.ExpiresAt != 0 && now >= claims.ExpiresAt {
		return nil, ports.ErrTokenExpired
	}

	if claims.Subject == "" {
		return nil, errors.New("security: sub (user_id) claim cannot be empty")
	}

	return &ports.UserIdentity{
		UserID:    claims.Subject,
		DeviceID:  claims.DeviceID,
		IssuedAt:  time.Unix(claims.IssuedAt, 0).UTC(),
		ExpiresAt: time.Unix(claims.ExpiresAt, 0).UTC(),
	}, nil
}

func preAuthEncode(pieces ...[]byte) []byte {
	var totalLen int
	totalLen += 8 // count header (uint64)
	for _, p := range pieces {
		totalLen += 8 + len(p)
	}

	buf := make([]byte, 0, totalLen)
	buf = appendLe64(buf, uint64(len(pieces)))
	for _, p := range pieces {
		buf = appendLe64(buf, uint64(len(p)))
		buf = append(buf, p...)
	}
	return buf
}

func appendLe64(dst []byte, n uint64) []byte {
	return append(dst,
		byte(n),
		byte(n>>8),
		byte(n>>16),
		byte(n>>24),
		byte(n>>32),
		byte(n>>40),
		byte(n>>48),
		byte(n>>56),
	)
}
