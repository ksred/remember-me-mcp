package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	// KeySize is the size of the encryption key in bytes (256 bits)
	KeySize = 32
	// NonceSize is the size of the GCM nonce in bytes
	NonceSize = 12
	// SaltSize is the size of the salt for key derivation
	SaltSize = 32
)

// EncryptionService handles field-level encryption for sensitive data
type EncryptionService struct {
	masterKey []byte
}

// NewEncryptionService creates a new encryption service with the provided master key
func NewEncryptionService(masterKeyBase64 string) (*EncryptionService, error) {
	if masterKeyBase64 == "" {
		return nil, errors.New("master key cannot be empty")
	}

	masterKey, err := base64.StdEncoding.DecodeString(masterKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid master key format: %w", err)
	}

	if len(masterKey) != KeySize {
		return nil, fmt.Errorf("master key must be %d bytes, got %d", KeySize, len(masterKey))
	}

	return &EncryptionService{
		masterKey: masterKey,
	}, nil
}

// EncryptedData contains all the components needed to decrypt data
type EncryptedData struct {
	Ciphertext    string `json:"ciphertext"`     // Base64 encoded encrypted data
	EncryptedKey  string `json:"encrypted_key"`  // Base64 encoded encrypted data key
	Nonce         string `json:"nonce"`          // Base64 encoded GCM nonce
	KeyNonce      string `json:"key_nonce"`      // Base64 encoded nonce for key encryption
}

// EncryptField encrypts a field value using a unique data key
func (s *EncryptionService) EncryptField(plaintext string) (*EncryptedData, error) {
	if plaintext == "" {
		return nil, errors.New("plaintext cannot be empty")
	}

	// Generate a unique data key for this field
	dataKey := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, dataKey); err != nil {
		return nil, fmt.Errorf("failed to generate data key: %w", err)
	}

	// Encrypt the data key with the master key
	encryptedKey, keyNonce, err := s.encryptDataKey(dataKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data key: %w", err)
	}

	// Create AES cipher with the data key
	block, err := aes.NewCipher(dataKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Clear the data key from memory
	for i := range dataKey {
		dataKey[i] = 0
	}

	return &EncryptedData{
		Ciphertext:   base64.StdEncoding.EncodeToString(ciphertext),
		EncryptedKey: base64.StdEncoding.EncodeToString(encryptedKey),
		Nonce:        base64.StdEncoding.EncodeToString(nonce),
		KeyNonce:     base64.StdEncoding.EncodeToString(keyNonce),
	}, nil
}

// DecryptField decrypts an encrypted field value
func (s *EncryptionService) DecryptField(data *EncryptedData) (string, error) {
	if data == nil {
		return "", errors.New("encrypted data cannot be nil")
	}

	// Decode base64 values
	ciphertext, err := base64.StdEncoding.DecodeString(data.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	encryptedKey, err := base64.StdEncoding.DecodeString(data.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted key: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(data.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	keyNonce, err := base64.StdEncoding.DecodeString(data.KeyNonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode key nonce: %w", err)
	}

	// Decrypt the data key
	dataKey, err := s.decryptDataKey(encryptedKey, keyNonce)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data key: %w", err)
	}

	// Create AES cipher with the data key
	block, err := aes.NewCipher(dataKey)
	if err != nil {
		// Clear the data key before returning
		for i := range dataKey {
			dataKey[i] = 0
		}
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		// Clear the data key before returning
		for i := range dataKey {
			dataKey[i] = 0
		}
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt the ciphertext
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Clear the data key before returning
		for i := range dataKey {
			dataKey[i] = 0
		}
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	// Clear the data key from memory
	for i := range dataKey {
		dataKey[i] = 0
	}

	return string(plaintext), nil
}

// encryptDataKey encrypts a data key using the master key
func (s *EncryptionService) encryptDataKey(dataKey []byte) ([]byte, []byte, error) {
	// Create AES cipher with master key
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data key
	encryptedKey := gcm.Seal(nil, nonce, dataKey, nil)

	return encryptedKey, nonce, nil
}

// decryptDataKey decrypts a data key using the master key
func (s *EncryptionService) decryptDataKey(encryptedKey, nonce []byte) ([]byte, error) {
	// Create AES cipher with master key
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt the data key
	dataKey, err := gcm.Open(nil, nonce, encryptedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return dataKey, nil
}

// DeriveKey derives a key from the master key using HKDF
func (s *EncryptionService) DeriveKey(salt []byte, info []byte) ([]byte, error) {
	hash := sha256.New
	hkdfReader := hkdf.New(hash, s.masterKey, salt, info)

	key := make([]byte, KeySize)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	return key, nil
}

// GenerateMasterKey generates a new random master key
func GenerateMasterKey() (string, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}