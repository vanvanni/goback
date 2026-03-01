package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecryptFileRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "test.txt")
	encryptedFile := filepath.Join(tmpDir, "test.txt.enc")
	decryptedFile := filepath.Join(tmpDir, "test.txt.dec")

	content := []byte("Hello, World! This is a test for encryption.")
	if err := os.WriteFile(inputFile, content, 0644); err != nil {
		t.Fatalf("Failed to write input file: %v", err)
	}

	key := "super-secret-key"

	if err := EncryptFile(inputFile, encryptedFile, key); err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	if err := DecryptFile(encryptedFile, decryptedFile, key); err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	decryptedData, err := os.ReadFile(decryptedFile)
	if err != nil {
		t.Fatalf("Failed to read decrypted file: %v", err)
	}

	if string(decryptedData) != string(content) {
		t.Errorf("Decrypted content does not match. Got %q, want %q", decryptedData, content)
	}
}

func TestDecryptFileWrongKey(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "test.txt")
	encryptedFile := filepath.Join(tmpDir, "test.txt.enc")
	decryptedFile := filepath.Join(tmpDir, "test.txt.dec")

	if err := os.WriteFile(inputFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write input file: %v", err)
	}

	if err := EncryptFile(inputFile, encryptedFile, "correct-key"); err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	err := DecryptFile(encryptedFile, decryptedFile, "wrong-key")
	if err != nil {
		return
	}
	t.Fatal("expected decryption to fail with wrong key")
}
