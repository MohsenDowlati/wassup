package security

import (
	"crypto/ed25519"
	"errors"

	"aidanwoods.dev/go-paseto"
	"github.com/MohsenDowlati/wassup/internal/core/ports"
)

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
	pub, err := paseto.NewV4AsymmetricPublicKeyFromEd25519(v.publicKey)
	if err != nil {
		return nil, ports.ErrTokenMalformed
	}
	parsed, err := paseto.NewParserForValidNow().ParseV4Public(pub, token, nil)
	if err != nil {
		return nil, ports.ErrTokenInvalidSig
	}
	subject, _ := parsed.GetSubject()
	if subject == "" {
		return nil, errors.New("security: sub (user_id) claim cannot be empty")
	}
	deviceID, _ := parsed.GetString("dev_id")
	issuedAt, _ := parsed.GetIssuedAt()
	expiresAt, _ := parsed.GetExpiration()

	return &ports.UserIdentity{
		UserID:    subject,
		DeviceID:  deviceID,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}, nil
}
