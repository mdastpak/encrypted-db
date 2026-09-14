package auth

import (
	"crypto/rsa"
	"sync"

	"github.com/golang-jwt/jwt/v5"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	keysOnce   sync.Once
	keysErr    error
)

// LoadKeys loads the RSA private and public keys once at startup.
// Thread-safe: uses sync.Once to ensure keys are loaded only once.
func LoadKeys() error {
	keysOnce.Do(func() {
		keysErr = loadKeysInternal()
	})
	return keysErr
}

func loadKeysInternal() error {
	// Keys should be loaded from config at startup, not per-request.
	// This function is kept for backward compatibility but should not be used.
	// Actual key loading happens in InitKeys() during application startup.
	return nil
}

// InitKeys initializes RSA keys from PEM data. Call once at application startup.
func InitKeys(privateKeyPEM, publicKeyPEM []byte) error {
	var err error
	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return err
	}
	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return err
	}
	return nil
}

// PrivateKey returns the loaded RSA private key.
func PrivateKey() *rsa.PrivateKey {
	return privateKey
}

// PublicKey returns the loaded RSA public key.
func PublicKey() *rsa.PublicKey {
	return publicKey
}

// KeysLoaded returns true if keys have been successfully initialized.
func KeysLoaded() bool {
	return privateKey != nil && publicKey != nil
}
