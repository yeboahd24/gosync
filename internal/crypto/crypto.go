package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
)

// Manager handles encryption and decryption operations
type Manager struct {
	key []byte
}

// NewManager creates a new crypto manager with the given key
func NewManager(keyFile string) (*Manager, error) {
	key, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("error reading key file: %w", err)
	}

	return &Manager{key: key}, nil
}

// EncryptFile encrypts the source file and writes to destination
func (m *Manager) EncryptFile(source, dest string) error {
	plaintext, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("error reading source file: %w", err)
	}

	block, err := aes.NewCipher(m.key)
	if err != nil {
		return fmt.Errorf("error creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("error creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("error generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	if err := os.WriteFile(dest, ciphertext, 0644); err != nil {
		return fmt.Errorf("error writing encrypted file: %w", err)
	}

	return nil
}

// DecryptFile decrypts the source file and writes to destination
func (m *Manager) DecryptFile(source, dest string) error {
	ciphertext, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("error reading encrypted file: %w", err)
	}

	block, err := aes.NewCipher(m.key)
	if err != nil {
		return fmt.Errorf("error creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("error creating GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("error decrypting file: %w", err)
	}

	if err := os.WriteFile(dest, plaintext, 0644); err != nil {
		return fmt.Errorf("error writing decrypted file: %w", err)
	}

	return nil
}
