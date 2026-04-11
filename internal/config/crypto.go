package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/denisbrodbeck/machineid"
)

const appID = "omnistrate-ctl"

// deriveKey returns a 32-byte AES-256 key derived from the machine ID.
func deriveKey() ([]byte, error) {
	id, err := machineid.ProtectedID(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get machine id: %w", err)
	}
	h := sha256.Sum256([]byte(id))
	return h[:], nil
}

// EncryptToken encrypts plaintext using AES-256-GCM with a machine-bound key.
// Returns a hex-encoded string of nonce+ciphertext.
func EncryptToken(plaintext string) (string, error) {
	key, err := deriveKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// DecryptToken decrypts a hex-encoded AES-256-GCM ciphertext using the machine-bound key.
func DecryptToken(encoded string) (string, error) {
	key, err := deriveKey()
	if err != nil {
		return "", err
	}

	data, err := hex.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode token: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	return string(plaintext), nil
}
