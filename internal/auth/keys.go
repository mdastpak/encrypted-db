package auth

import (
	"crypto/rsa"
	"encrypted-db/config"
	"fmt"
	"log"
	"os"

	"github.com/golang-jwt/jwt/v4"
)

// Global variables for RSA keys
var privateKey *rsa.PrivateKey
var publicKey *rsa.PublicKey

// LoadKeys loads the RSA private and public keys from internal/ssl directory
func LoadKeys() error {
	// Load private key from internal/ssl directory
	privKeyData, err := os.ReadFile(config.Config.JWT.PrivateKey) // Updated path to internal/ssl
	if err != nil {

		return fmt.Errorf("failed to load private key: %v", err)
	}
	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	// Load public key from internal/ssl directory
	pubKeyData, err := os.ReadFile(config.Config.JWT.PublicKey) // Updated path to internal/ssl
	if err != nil {
		return fmt.Errorf("failed to load public key: %v", err)
	}
	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(pubKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %v", err)
	}

	log.Println("RSA keys loaded successfully")
	return nil
}
