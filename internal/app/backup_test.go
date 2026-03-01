package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vanvanni/goback/internal/drivers"
	"github.com/vanvanni/goback/internal/helper"
	"github.com/vanvanni/goback/internal/spec"
)

func TestBackupTaskRunCreatesArchive(t *testing.T) {
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "hello.txt")
	if err := os.WriteFile(sourceFile, []byte("hello backup"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	task := BackupTask{
		Definition: &spec.BackupDefinition{
			FileName: fmt.Sprintf("plain-%d", time.Now().UnixNano()),
		},
		Directory: []drivers.DirectoryDriver{
			{
				Name:   "data",
				Source: sourceDir,
			},
		},
	}

	archive, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("BackupTask.Run failed: %v", err)
	}
	defer func() { _ = os.Remove(archive) }()

	if !strings.HasSuffix(archive, ".tar.gz") {
		t.Fatalf("expected .tar.gz archive, got %q", archive)
	}

	info, err := os.Stat(archive)
	if err != nil {
		t.Fatalf("expected archive file to exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected non-empty archive")
	}
}

func TestBackupTaskRunCreatesEncryptedArchive(t *testing.T) {
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "hello.txt")
	if err := os.WriteFile(sourceFile, []byte("hello encrypted backup"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	task := BackupTask{
		Definition: &spec.BackupDefinition{
			FileName: fmt.Sprintf("encrypted-%d", time.Now().UnixNano()),
		},
		Directory: []drivers.DirectoryDriver{
			{
				Name:   "data",
				Source: sourceDir,
			},
		},
		GlobalEncryptionKey: "super-secret",
	}

	archive, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("BackupTask.Run failed: %v", err)
	}
	defer func() { _ = os.Remove(archive) }()

	if !strings.HasSuffix(archive, ".enc") {
		t.Fatalf("expected .enc archive, got %q", archive)
	}

	decryptedPath := archive + ".dec"
	defer func() { _ = os.Remove(decryptedPath) }()

	if err := helper.DecryptFile(archive, decryptedPath, "super-secret"); err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	decryptedData, err := os.ReadFile(decryptedPath)
	if err != nil {
		t.Fatalf("failed to read decrypted archive: %v", err)
	}

	if len(decryptedData) < 2 || decryptedData[0] != 0x1f || decryptedData[1] != 0x8b {
		t.Fatalf("expected decrypted archive to be gzip data, got prefix %v", decryptedData[:min(2, len(decryptedData))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
