package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// EncryptedPrefix marks encrypted values in the database
	EncryptedPrefix = "enc:v1:"

	// Key derivation parameters
	pbkdf2Iterations = 100000
	keyLength        = 32 // AES-256
	saltLength       = 16
)

var (
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrDecryptionFailed  = errors.New("decryption failed")
)

type SecretStore struct {
	key []byte
}

// NewSecretStore creates a new secret store with a key derived from the admin PIN.
// The salt should be stored persistently (e.g., in the settings table).
func NewSecretStore(pin string, salt []byte) *SecretStore {
	key := pbkdf2.Key([]byte(pin), salt, pbkdf2Iterations, keyLength, sha256.New)
	return &SecretStore{key: key}
}

// GenerateSalt creates a random salt for key derivation.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// Encrypt encrypts a plaintext string using AES-256-GCM.
// Returns a base64-encoded ciphertext with the EncryptedPrefix.
func (s *SecretStore) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return EncryptedPrefix + encoded, nil
}

// Decrypt decrypts a ciphertext that was encrypted with Encrypt.
// If the value doesn't have the EncryptedPrefix, it's returned as-is (for migration).
func (s *SecretStore) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// If not encrypted, return as-is (handles unencrypted legacy data)
	if len(ciphertext) < len(EncryptedPrefix) || ciphertext[:len(EncryptedPrefix)] != EncryptedPrefix {
		return ciphertext, nil
	}

	encoded := ciphertext[len(EncryptedPrefix):]
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a value has the encryption prefix.
func IsEncrypted(value string) bool {
	return len(value) >= len(EncryptedPrefix) && value[:len(EncryptedPrefix)] == EncryptedPrefix
}

// MustDecrypt decrypts a value, returning the original if decryption fails.
// Useful for graceful degradation during migration.
func (s *SecretStore) MustDecrypt(ciphertext string) string {
	plaintext, err := s.Decrypt(ciphertext)
	if err != nil {
		return ciphertext
	}
	return plaintext
}
