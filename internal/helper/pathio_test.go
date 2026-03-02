package helper

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSplitPath(t *testing.T) {
	dir, name, err := splitPath("file.txt")
	if err != nil {
		t.Fatalf("splitPath returned error: %v", err)
	}
	if dir != "." {
		t.Fatalf("expected dir '.', got %q", dir)
	}
	if name != "file.txt" {
		t.Fatalf("expected name 'file.txt', got %q", name)
	}

	_, _, err = splitPath("   ")
	if err == nil {
		t.Fatal("expected error for empty/whitespace path")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("unexpected error: %v", err)
	}

	rootPath := "/"
	if runtime.GOOS == "windows" {
		rootPath = filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	}

	_, _, err = splitPath(rootPath)
	if err == nil {
		t.Fatal("expected error for root-only path")
	}
	if !strings.Contains(err.Error(), "must reference a file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPathIOReadWriteOpenRoundTrip(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "payload.txt")
	initial := []byte("hello")
	updated := []byte("updated")

	if err := WriteFileSafe(filePath, initial, 0600); err != nil {
		t.Fatalf("WriteFileSafe failed: %v", err)
	}

	got, err := ReadFileSafe(filePath)
	if err != nil {
		t.Fatalf("ReadFileSafe failed: %v", err)
	}
	if string(got) != string(initial) {
		t.Fatalf("unexpected file contents: got %q want %q", got, initial)
	}

	writeOnlyFile, err := OpenWriteOnlyFile(filePath, 0600)
	if err != nil {
		t.Fatalf("OpenWriteOnlyFile failed: %v", err)
	}
	if _, err := writeOnlyFile.Write(updated); err != nil {
		_ = writeOnlyFile.Close()
		t.Fatalf("failed to write updated payload: %v", err)
	}
	if err := writeOnlyFile.Close(); err != nil {
		t.Fatalf("failed to close write file: %v", err)
	}

	readOnlyFile, err := OpenReadOnlyFile(filePath)
	if err != nil {
		t.Fatalf("OpenReadOnlyFile failed: %v", err)
	}
	defer readOnlyFile.Close()

	data, err := io.ReadAll(readOnlyFile)
	if err != nil {
		t.Fatalf("failed to read from opened file: %v", err)
	}
	if string(data) != string(updated) {
		t.Fatalf("unexpected updated file contents: got %q want %q", data, updated)
	}
}

func TestPathIORejectsEmptyPath(t *testing.T) {
	if _, err := ReadFileSafe(""); err == nil {
		t.Fatal("expected ReadFileSafe to fail for empty path")
	}
	if err := WriteFileSafe("", []byte("x"), 0600); err == nil {
		t.Fatal("expected WriteFileSafe to fail for empty path")
	}
	if _, err := OpenReadOnlyFile(""); err == nil {
		t.Fatal("expected OpenReadOnlyFile to fail for empty path")
	}
	if _, err := OpenWriteOnlyFile("", 0600); err == nil {
		t.Fatal("expected OpenWriteOnlyFile to fail for empty path")
	}
}

func TestOpenReadOnlyFileReturnsErrorForMissingPath(t *testing.T) {
	_, err := OpenReadOnlyFile(filepath.Join(t.TempDir(), "missing.txt"))
	if err == nil {
		t.Fatal("expected OpenReadOnlyFile to fail for missing file")
	}
}
