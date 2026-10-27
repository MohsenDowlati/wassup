package main

import (
	"encoding/hex"
	"fmt"
	"time"

	"aidanwoods.dev/go-paseto"
)

func main() {
	// 1. Generate a new asymmetric keypair
	secretKey := paseto.NewV4AsymmetricSecretKey()
	publicKey := secretKey.Public()

	// Convert public key to hex for server config
	pubBytes := publicKey.ExportBytes()
	pubHex := hex.EncodeToString(pubBytes)

	issuer := "wassup-auth-service"
	audience := "wassup-gateway"

	fmt.Println("==================================================")
	fmt.Println("👉 Put this in your run.bat as PASETO_PUBLIC_KEY_HEX:")
	fmt.Printf("set PASETO_PUBLIC_KEY_HEX=%s\n", pubHex)
	fmt.Println("==================================================")

	// 2. Generate token for user_alice
	aliceToken := createToken(secretKey, "user_alice", issuer, audience)
	fmt.Printf("\n🔑 Token for user_alice:\n%s\n", aliceToken)

	// 3. Generate token for user_bob
	bobToken := createToken(secretKey, "user_bob", issuer, audience)
	fmt.Printf("\n🔑 Token for user_bob:\n%s\n", bobToken)
	fmt.Println("==================================================")
}

func createToken(key paseto.V4AsymmetricSecretKey, userID, issuer, audience string) string {
	token := paseto.NewToken()
	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())
	token.SetExpiration(time.Now().Add(24 * time.Hour))
	token.SetIssuer(issuer)
	token.SetAudience(audience)
	token.SetSubject(userID) // Subject matches Sender User ID

	return token.V4Sign(key, nil)
}
