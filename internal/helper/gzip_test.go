package helper

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompressFileRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dest := filepath.Join(tmpDir, "source.txt.gz")
	content := "hello from gzip test"

	if err := os.WriteFile(src, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	if err := CompressFile(src, dest); err != nil {
		t.Fatalf("CompressFile failed: %v", err)
	}

	f, err := os.Open(dest)
	if err != nil {
		t.Fatalf("failed to open gzip output: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	got, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("failed to read decompressed output: %v", err)
	}

	if string(got) != content {
		t.Fatalf("expected %q, got %q", content, string(got))
	}
}

func TestCompressFileMissingSource(t *testing.T) {
	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "output.gz")

	err := CompressFile(filepath.Join(tmpDir, "missing.txt"), dest)
	if err == nil {
		t.Fatal("expected error when source file does not exist")
	}
}

func TestCompressDirCreatesArchiveWithContents(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	dest := filepath.Join(tmpDir, "archive.tar.gz")

	if err := os.MkdirAll(filepath.Join(srcDir, "sub"), 0755); err != nil {
		t.Fatalf("failed to create source subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("A"), 0644); err != nil {
		t.Fatalf("failed to write a.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "sub", "b.txt"), []byte("B"), 0644); err != nil {
		t.Fatalf("failed to write b.txt: %v", err)
	}

	if err := CompressDir(srcDir, dest); err != nil {
		t.Fatalf("CompressDir failed: %v", err)
	}

	f, err := os.Open(dest)
	if err != nil {
		t.Fatalf("failed to open archive: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	files := map[string]string{}
	entries := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed reading tar stream: %v", err)
		}

		entries[hdr.Name] = true
		if hdr.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(tr)
			if err != nil {
				t.Fatalf("failed to read file %q from tar: %v", hdr.Name, err)
			}
			files[hdr.Name] = string(data)
		}
	}

	if files["a.txt"] != "A" {
		t.Fatalf("expected a.txt content %q, got %q", "A", files["a.txt"])
	}
	if files["sub/b.txt"] != "B" {
		t.Fatalf("expected sub/b.txt content %q, got %q", "B", files["sub/b.txt"])
	}

	hasSubDir := entries["sub"] || entries["sub/"]
	if !hasSubDir {
		all := make([]string, 0, len(entries))
		for name := range entries {
			all = append(all, name)
		}
		t.Fatalf("expected sub directory entry, got entries: %s", strings.Join(all, ", "))
	}
}
