package helper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

func EncryptFile(inputPath, outputPath, key string) error {
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Generate a random salt
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using PBKDF2
	dk := pbkdf2.Key([]byte(key), salt, 4096, 32, sha256.New)

	block, err := aes.NewCipher(dk)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Combine salt + nonce + ciphertext
	// We need to store salt and nonce to decrypt later
	finalData := append(salt, nonce...)
	finalData = append(finalData, ciphertext...)

	if err := os.WriteFile(outputPath, finalData, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}
