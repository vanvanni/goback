package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDirectoryStructureCreatesPaths(t *testing.T) {
	tmpDir := t.TempDir()

	if err := EnsureDirectoryStructure(tmpDir); err != nil {
		t.Fatalf("EnsureDirectoryStructure failed: %v", err)
	}

	wantSpecDir := filepath.Join(tmpDir, "specs.d")
	if got := GetSpecDirectory(); got != wantSpecDir {
		t.Fatalf("expected spec directory %q, got %q", wantSpecDir, got)
	}
	if _, err := os.Stat(wantSpecDir); err != nil {
		t.Fatalf("expected spec directory to exist: %v", err)
	}

	configPath := filepath.Join(tmpDir, "config.yml")
	if got := GetCoreConfigPath(); got != configPath {
		t.Fatalf("expected config path %q, got %q", configPath, got)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
}

func TestEnsureDirectoryStructureDoesNotOverwriteConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	original := []byte("encryption_key: existing-key\n")

	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}

	if err := EnsureDirectoryStructure(tmpDir); err != nil {
		t.Fatalf("EnsureDirectoryStructure failed: %v", err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("expected config to be preserved, got %q", string(got))
	}
}
