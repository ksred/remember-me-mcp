package utils

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateMasterKey(t *testing.T) {
	key, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	// Check that key is base64 encoded
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Fatalf("Generated key is not valid base64: %v", err)
	}

	// Check key size
	if len(decoded) != KeySize {
		t.Errorf("Expected key size %d, got %d", KeySize, len(decoded))
	}
}

func TestNewEncryptionService(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantError bool
	}{
		{
			name:      "valid key",
			key:       base64.StdEncoding.EncodeToString(make([]byte, KeySize)),
			wantError: false,
		},
		{
			name:      "empty key",
			key:       "",
			wantError: true,
		},
		{
			name:      "invalid base64",
			key:       "not-base64!@#$",
			wantError: true,
		},
		{
			name:      "wrong key size",
			key:       base64.StdEncoding.EncodeToString(make([]byte, 16)),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncryptionService(tt.key)
			if (err != nil) != tt.wantError {
				t.Errorf("NewEncryptionService() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	// Generate a test master key
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	service, err := NewEncryptionService(masterKey)
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
		wantError bool
	}{
		{
			name:      "simple text",
			plaintext: "Hello, World!",
			wantError: false,
		},
		{
			name:      "long text",
			plaintext: strings.Repeat("This is a long message. ", 100),
			wantError: false,
		},
		{
			name:      "unicode text",
			plaintext: "Hello 世界 🌍 Здравствуй мир",
			wantError: false,
		},
		{
			name:      "empty text",
			plaintext: "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := service.EncryptField(tt.plaintext)
			if (err != nil) != tt.wantError {
				t.Errorf("EncryptField() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				return
			}

			// Verify encrypted data structure
			if encrypted.Ciphertext == "" {
				t.Error("Ciphertext is empty")
			}
			if encrypted.EncryptedKey == "" {
				t.Error("EncryptedKey is empty")
			}
			if encrypted.Nonce == "" {
				t.Error("Nonce is empty")
			}
			if encrypted.KeyNonce == "" {
				t.Error("KeyNonce is empty")
			}

			// Decrypt
			decrypted, err := service.DecryptField(encrypted)
			if err != nil {
				t.Errorf("DecryptField() error = %v", err)
				return
			}

			// Verify decryption
			if decrypted != tt.plaintext {
				t.Errorf("Decrypted text doesn't match. Got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptionUniqueness(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	service, err := NewEncryptionService(masterKey)
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	plaintext := "Test message"

	// Encrypt the same message multiple times
	encrypted1, err := service.EncryptField(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	encrypted2, err := service.EncryptField(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Verify that encrypted data is different each time
	if encrypted1.Ciphertext == encrypted2.Ciphertext {
		t.Error("Ciphertext should be different for each encryption")
	}
	if encrypted1.EncryptedKey == encrypted2.EncryptedKey {
		t.Error("EncryptedKey should be different for each encryption")
	}
	if encrypted1.Nonce == encrypted2.Nonce {
		t.Error("Nonce should be different for each encryption")
	}
}

func TestDecryptWithInvalidData(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	service, err := NewEncryptionService(masterKey)
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	// Create valid encrypted data first
	encrypted, err := service.EncryptField("Test message")
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	tests := []struct {
		name string
		data *EncryptedData
	}{
		{
			name: "nil data",
			data: nil,
		},
		{
			name: "tampered ciphertext",
			data: &EncryptedData{
				Ciphertext:   "tampered" + encrypted.Ciphertext,
				EncryptedKey: encrypted.EncryptedKey,
				Nonce:        encrypted.Nonce,
				KeyNonce:     encrypted.KeyNonce,
			},
		},
		{
			name: "tampered key",
			data: &EncryptedData{
				Ciphertext:   encrypted.Ciphertext,
				EncryptedKey: "tampered" + encrypted.EncryptedKey,
				Nonce:        encrypted.Nonce,
				KeyNonce:     encrypted.KeyNonce,
			},
		},
		{
			name: "invalid base64",
			data: &EncryptedData{
				Ciphertext:   "not-base64!@#$",
				EncryptedKey: encrypted.EncryptedKey,
				Nonce:        encrypted.Nonce,
				KeyNonce:     encrypted.KeyNonce,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.DecryptField(tt.data)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestDeriveKey(t *testing.T) {
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	service, err := NewEncryptionService(masterKey)
	if err != nil {
		t.Fatalf("Failed to create encryption service: %v", err)
	}

	salt := make([]byte, SaltSize)
	info := []byte("test context")

	key1, err := service.DeriveKey(salt, info)
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}

	if len(key1) != KeySize {
		t.Errorf("Expected key size %d, got %d", KeySize, len(key1))
	}

	// Same salt and info should produce same key
	key2, err := service.DeriveKey(salt, info)
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}

	if string(key1) != string(key2) {
		t.Error("Same salt and info should produce same derived key")
	}

	// Different salt should produce different key
	salt2 := make([]byte, SaltSize)
	salt2[0] = 1 // Make it different
	key3, err := service.DeriveKey(salt2, info)
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}

	if string(key1) == string(key3) {
		t.Error("Different salt should produce different derived key")
	}
}