package helper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltSize = 16
)

func EncryptFile(inputPath, outputPath, key string) error {
	if key == "" {
		return fmt.Errorf("encryption key is required")
	}

	plaintext, err := ReadFileSafe(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Generate a random salt
	salt := make([]byte, saltSize)
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

	if err := WriteFileSafe(outputPath, finalData, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}

func DecryptFile(inputPath, outputPath, key string) error {
	if key == "" {
		return fmt.Errorf("decryption key is required")
	}

	encryptedData, err := ReadFileSafe(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}
	if len(encryptedData) < saltSize {
		return fmt.Errorf("encrypted file is too short")
	}

	salt := encryptedData[:saltSize]
	dk := pbkdf2.Key([]byte(key), salt, 4096, 32, sha256.New)
	block, err := aes.NewCipher(dk)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(encryptedData) < saltSize+gcm.NonceSize() {
		return fmt.Errorf("encrypted file is too short")
	}

	nonce := encryptedData[saltSize : saltSize+gcm.NonceSize()]
	ciphertext := encryptedData[saltSize+gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	if err := WriteFileSafe(outputPath, plaintext, 0600); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	return nil
}
