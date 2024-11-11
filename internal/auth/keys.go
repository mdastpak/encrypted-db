package auth

import (
	"crypto/rsa"
	"encrypted-db/config"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-jwt/jwt/v4"
)

// Global variables for RSA keys
var privateKey *rsa.PrivateKey
var publicKey *rsa.PublicKey

// LoadKeys loads the RSA private and public keys from internal/ssl directory
func LoadKeys() error {

	// Build the path to the private key file relative to the root
	// privateKeyPath := filepath.Join(projectRoot, "internal", "ssl", "user", "private_key.pem")
	privateKeyPath := filepath.Join(config.Config.JWT.SSL.User.PrivateKey...) // Updated path to internal/ssl
	log.Printf("Private key path: %s\n", privateKeyPath)

	// Load private key from internal/ssl directory
	privKeyData, err := os.ReadFile(privateKeyPath) // Updated path to internal/ssl
	if err != nil {
		return fmt.Errorf("failed to load private key: %v", err)
	}
	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	publicKeyPath := filepath.Join(config.Config.JWT.SSL.User.PublicKey...) // Updated path to internal/ssl
	log.Printf("Public key path: %s\n", publicKeyPath)
	// Load public key from internal/ssl directory
	pubKeyData, err := os.ReadFile(publicKeyPath) // Updated path to internal/ssl
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
