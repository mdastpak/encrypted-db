package public

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
)

// EncryptWithAES encrypts the given plaintext with the provided key and salt (UUID)
func EncryptWithAES(plaintext, key string) (string, error) {
	log.Println("Starting encryption process")

	// Adjust key to be exactly 32 bytes for AES-256
	if len(key) < 32 {
		log.Println("Extending key length to 32 bytes with padding")
		key = key + string(make([]byte, 32-len(key))) // Pad with zeros if key is too short
	} else if len(key) > 32 {
		log.Println("Trimming key length to 32 bytes")
		key = key[:32]
	}
	log.Printf("Encryption key: %s\n", key)

	// Create AES block cipher
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		log.Printf("Error creating AES cipher block: %v\n", err)
		return "", err
	}
	log.Println("AES cipher block created successfully")

	// Use GCM for encryption mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Printf("Error creating GCM block: %v\n", err)
		return "", err
	}
	log.Println("GCM block created successfully")

	// Create a nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Printf("Error generating nonce: %v\n", err)
		return "", err
	}
	log.Printf("Nonce generated: %x\n", nonce)

	// Encrypt the data and prepend the nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encryptedText := hex.EncodeToString(ciphertext)
	log.Printf("Encryption complete: %s\n", encryptedText)

	return encryptedText, nil
}

// DecryptWithAES decrypts the given ciphertext with the provided key and salt (UUID)
func DecryptWithAES(ciphertext, key string) (string, error) {
	log.Println("Starting decryption process")

	// Adjust key to be exactly 32 bytes for AES-256
	if len(key) < 32 {
		log.Println("Extending key length to 32 bytes with padding")
		key = key + string(make([]byte, 32-len(key))) // Pad with zeros if key is too short
	} else if len(key) > 32 {
		log.Println("Trimming key length to 32 bytes")
		key = key[:32]
	}
	log.Printf("Decryption key: %s\n", key)

	// Decode the hex-encoded ciphertext
	data, err := hex.DecodeString(ciphertext)
	if err != nil {
		log.Printf("Error decoding ciphertext: %v\n", err)
		return "", err
	}
	log.Printf("Decoded ciphertext: %x\n", data)

	// Create AES block cipher
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		log.Printf("Error creating AES cipher block: %v\n", err)
		return "", err
	}
	log.Println("AES cipher block created successfully")

	// Use GCM for decryption mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Printf("Error creating GCM block: %v\n", err)
		return "", err
	}
	log.Println("GCM block created successfully")

	// Extract nonce and actual ciphertext
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		log.Println("Invalid ciphertext: insufficient length for nonce")
		return "", errors.New("invalid ciphertext")
	}
	nonce, ciphertextData := data[:nonceSize], data[nonceSize:]
	log.Printf("Nonce extracted: %x\n", nonce)
	log.Printf("Ciphertext data extracted: %x\n", ciphertextData)

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertextData, nil)
	if err != nil {
		log.Printf("Error during decryption: %v\n", err)
		return "", err
	}
	log.Printf("Decryption complete: %s\n", plaintext)

	return string(plaintext), nil
}
