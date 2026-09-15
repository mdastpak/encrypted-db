package auth

import (
	"crypto/rsa"
	"sync"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// Current active keys
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey

	// Key rotation: map of key ID -> public key for verifying old tokens
	publicKeys map[string]*rsa.PublicKey

	keysOnce   sync.Once
	keysErr    error
	keysMutex  sync.RWMutex
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
	publicKeys = make(map[string]*rsa.PublicKey)
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

	keysMutex.Lock()
	defer keysMutex.Unlock()

	if publicKeys == nil {
		publicKeys = make(map[string]*rsa.PublicKey)
	}
	// Default key ID for current key
	publicKeys["current"] = publicKey
	return nil
}

// AddPublicKey adds a public key for token verification (key rotation support).
// keyID is a unique identifier for the key (e.g., "key-2024-01", "key-2024-02").
func AddPublicKey(keyID string, publicKeyPEM []byte) error {
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return err
	}

	keysMutex.Lock()
	defer keysMutex.Unlock()

	if publicKeys == nil {
		publicKeys = make(map[string]*rsa.PublicKey)
	}
	publicKeys[keyID] = pubKey
	return nil
}

// RemovePublicKey removes a public key (e.g., when a key is fully retired).
func RemovePublicKey(keyID string) {
	keysMutex.Lock()
	defer keysMutex.Unlock()
	delete(publicKeys, keyID)
}

// GetPublicKeys returns all registered public keys for verification.
func GetPublicKeys() map[string]*rsa.PublicKey {
	keysMutex.RLock()
	defer keysMutex.RUnlock()

	result := make(map[string]*rsa.PublicKey, len(publicKeys))
	for k, v := range publicKeys {
		result[k] = v
	}
	return result
}

// PrivateKey returns the current active RSA private key (for signing).
func PrivateKey() *rsa.PrivateKey {
	return privateKey
}

// PublicKey returns the current active RSA public key (for verification).
func PublicKey() *rsa.PublicKey {
	return publicKey
}

// KeysLoaded returns true if keys have been successfully initialized.
func KeysLoaded() bool {
	return privateKey != nil && publicKey != nil
}

// VerifyRSATokenWithKeys verifies a JWT using all registered public keys (supports key rotation).
func VerifyRSATokenWithKeys(tokenString string) (*Claims, error) {
	if !KeysLoaded() {
		return nil, ErrKeysNotLoaded
	}

	claims := &Claims{}
	keys := GetPublicKeys()

	var lastErr error
	for _, pubKey := range keys {
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, ErrInvalidSigningAlg
			}
			return pubKey, nil
		})

		if err == nil && token.Valid {
			// Token validated successfully with this key
			return claims, nil
		}
		lastErr = err
		// Continue trying other keys
	}

	return nil, lastErr
}
