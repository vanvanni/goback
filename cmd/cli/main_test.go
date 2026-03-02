package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultDecryptOutput(t *testing.T) {
	if got := defaultDecryptOutput("backup.tar.gz.enc"); got != "backup.tar.gz" {
		t.Fatalf("expected stripped .enc suffix, got %q", got)
	}

	if got := defaultDecryptOutput("backup.tar.gz"); got != "backup.tar.gz.dec" {
		t.Fatalf("expected .dec suffix, got %q", got)
	}
}

func TestResolveSpecPath(t *testing.T) {
	baseDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(baseDir, "specs.d"), 0755)
	specPath := writeValidSpec(t, baseDir, "daily.yml", "")

	got, err := resolveSpecPath(baseDir, "daily")
	if err != nil {
		t.Fatalf("resolveSpecPath by name failed: %v", err)
	}
	if got != specPath {
		t.Fatalf("expected %q, got %q", specPath, got)
	}

	got, err = resolveSpecPath(baseDir, "daily.yml")
	if err != nil {
		t.Fatalf("resolveSpecPath by name with extension failed: %v", err)
	}
	if got != specPath {
		t.Fatalf("expected %q, got %q", specPath, got)
	}

	got, err = resolveSpecPath(baseDir, specPath)
	if err != nil {
		t.Fatalf("resolveSpecPath by absolute path failed: %v", err)
	}
	if got != specPath {
		t.Fatalf("expected %q, got %q", specPath, got)
	}
}

func TestResolveDecryptKeyPrecedence(t *testing.T) {
	baseDir := t.TempDir()
	writeConfig(t, baseDir, "global-key")
	writeValidSpec(t, baseDir, "daily.yml", "spec-key")

	key, err := resolveDecryptKey("cli-key", baseDir, "daily.yml")
	if err != nil {
		t.Fatalf("resolveDecryptKey with explicit key failed: %v", err)
	}
	if key != "cli-key" {
		t.Fatalf("expected explicit key, got %q", key)
	}

	key, err = resolveDecryptKey("", baseDir, "daily.yml")
	if err != nil {
		t.Fatalf("resolveDecryptKey with spec key failed: %v", err)
	}
	if key != "spec-key" {
		t.Fatalf("expected spec key, got %q", key)
	}

	key, err = resolveDecryptKey("", baseDir, "")
	if err != nil {
		t.Fatalf("resolveDecryptKey with config key failed: %v", err)
	}
	if key != "global-key" {
		t.Fatalf("expected config key, got %q", key)
	}
}

func TestResolveDecryptKeyMissing(t *testing.T) {
	baseDir := t.TempDir()
	writeConfig(t, baseDir, "")

	_, err := resolveDecryptKey("", baseDir, "")
	if err == nil {
		t.Fatal("expected error when no key is available")
	}
	if !strings.Contains(err.Error(), "no decryption key found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveDecryptKeyRequiresDirWhenNoExplicitKey(t *testing.T) {
	_, err := resolveDecryptKey("", "", "")
	if err == nil {
		t.Fatal("expected error when dir is missing and key is not provided")
	}
	if !strings.Contains(err.Error(), "missing decryption key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveDecryptKeySpecNotFound(t *testing.T) {
	baseDir := t.TempDir()
	writeConfig(t, baseDir, "global-key")

	_, err := resolveDecryptKey("", baseDir, "does-not-exist")
	if err == nil {
		t.Fatal("expected error when spec does not exist")
	}
	if !strings.Contains(err.Error(), "was not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveDecryptKeyFallsBackToConfigWhenSpecHasNoKey(t *testing.T) {
	baseDir := t.TempDir()
	writeConfig(t, baseDir, "global-key")
	writeValidSpec(t, baseDir, "daily.yml", "")

	key, err := resolveDecryptKey("", baseDir, "daily.yml")
	if err != nil {
		t.Fatalf("resolveDecryptKey failed: %v", err)
	}
	if key != "global-key" {
		t.Fatalf("expected fallback to config key, got %q", key)
	}
}

func writeConfig(t *testing.T, baseDir, key string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(baseDir, "specs.d"), 0755); err != nil {
		t.Fatalf("failed to create specs dir: %v", err)
	}

	content := ""
	if key != "" {
		content = fmt.Sprintf("encryption_key: %q\n", key)
	}

	if err := os.WriteFile(filepath.Join(baseDir, "config.yml"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

func writeValidSpec(t *testing.T, baseDir, name, encryptionKey string) string {
	t.Helper()

	sourceDir := filepath.Join(baseDir, "data")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	specContent := fmt.Sprintf(`interval: 10s
repositories:
  - source: s3:main
    dest: backups
directory:
  - name: data
    source: %q
`, filepath.ToSlash(sourceDir))

	if encryptionKey != "" {
		specContent += fmt.Sprintf("encryption_key: %q\n", encryptionKey)
	}

	specPath := filepath.Join(baseDir, "specs.d", name)
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	return specPath
}
