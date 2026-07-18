package helper

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTarGz(t *testing.T) {
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	archive := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := CompressDir(sourceDir, archive); err != nil {
		t.Fatalf("CompressDir failed: %v", err)
	}

	dest := t.TempDir()
	if err := ExtractTarGz(archive, dest); err != nil {
		t.Fatalf("ExtractTarGz failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "hello.txt"))
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected %q, got %q", "hello", string(data))
	}
}

func TestExtractGzipFile(t *testing.T) {
	source := filepath.Join(t.TempDir(), "dump.sql")
	if err := os.WriteFile(source, []byte("select 1;"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	archive := filepath.Join(t.TempDir(), "dump.sql.gz")
	if err := CompressFile(source, archive); err != nil {
		t.Fatalf("CompressFile failed: %v", err)
	}

	target := filepath.Join(t.TempDir(), "out", "dump.sql")
	if err := ExtractGzipFile(archive, target); err != nil {
		t.Fatalf("ExtractGzipFile failed: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(data) != "select 1;" {
		t.Fatalf("expected %q, got %q", "select 1;", string(data))
	}
}

func TestExtractTarGzRejectsTraversal(t *testing.T) {
	sourceDir := t.TempDir()
	archive := filepath.Join(sourceDir, "bad.tar.gz")
	if err := os.WriteFile(archive, traversalArchive(t), 0600); err != nil {
		t.Fatalf("failed to write archive: %v", err)
	}

	err := ExtractTarGz(archive, t.TempDir())
	if err == nil {
		t.Fatal("expected traversal error")
	}
}

func traversalArchive(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	payload := []byte("bad")
	hdr := &tar.Header{
		Name: "../evil.txt",
		Mode: 0600,
		Size: int64(len(payload)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatalf("failed to write tar payload: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	return buf.Bytes()
}
