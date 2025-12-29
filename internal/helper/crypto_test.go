package helper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

func TestEncryptFile(t *testing.T) {
	// Create a temporary input file
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "test.txt")
	encryptedFile := filepath.Join(tmpDir, "test.txt.enc")

	content := []byte("Hello, World! This is a test for encryption.")
	if err := os.WriteFile(inputFile, content, 0644); err != nil {
		t.Fatalf("Failed to write input file: %v", err)
	}

	key := "super-secret-key"

	// Encrypt the file
	if err := EncryptFile(inputFile, encryptedFile, key); err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	// Verify encrypted file exists
	encryptedData, err := os.ReadFile(encryptedFile)
	if err != nil {
		t.Fatalf("Failed to read encrypted file: %v", err)
	}

	if len(encryptedData) <= len(content) {
		t.Errorf("Encrypted file should be larger than input file due to salt and nonce")
	}

	// Try to decrypt manually to verify structure
	salt := encryptedData[:16]
	dk := pbkdf2.Key([]byte(key), salt, 4096, 32, sha256.New)

	block, err := aes.NewCipher(dk)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("Failed to create GCM: %v", err)
	}

	nonceSize := gcm.NonceSize()
	nonce := encryptedData[16 : 16+nonceSize]
	ciphertext := encryptedData[16+nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(plaintext) != string(content) {
		t.Errorf("Decrypted content does not match. Got %q, want %q", plaintext, content)
	}
}
