package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigParsesEncryptionKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	configContent := `encryption_key: "global-key"
mariadb:
  host: "db.internal"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.EncryptionKey != "global-key" {
		t.Fatalf("expected encryption key %q, got %q", "global-key", cfg.EncryptionKey)
	}
	if cfg.MariaDB.Host != "db.internal" {
		t.Fatalf("expected mariadb host %q, got %q", "db.internal", cfg.MariaDB.Host)
	}
}

func TestLoadSpecValid(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "backup.yml")
	sourceDir := filepath.Join(tmpDir, "data")
	_ = os.MkdirAll(sourceDir, 0755)

	specContent := fmt.Sprintf(`interval: 1m
repositories:
  - source: s3:main
    dest: backups
directory:
  - name: data
    source: %q
encryption_key: "spec-key"
`, filepath.ToSlash(sourceDir))

	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	def, err := LoadSpec(specPath)
	if err != nil {
		t.Fatalf("LoadSpec failed: %v", err)
	}

	if len(def.Repositories) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(def.Repositories))
	}
	if len(def.Directory) != 1 {
		t.Fatalf("expected 1 directory driver, got %d", len(def.Directory))
	}
	if def.ScheduleTimer == nil {
		t.Fatal("expected schedule timer to be set")
	}
	if def.EncryptionKey != "spec-key" {
		t.Fatalf("expected encryption key %q, got %q", "spec-key", def.EncryptionKey)
	}
}

func TestLoadSpecValidationErrors(t *testing.T) {
	testCases := []struct {
		name       string
		content    string
		errSnippet string
	}{
		{
			name: "missing repositories",
			content: `interval: 1m
directory:
  - name: data
    source: "/tmp/data"
`,
			errSnippet: "at least one repository is required",
		},
		{
			name: "missing schedule",
			content: `repositories:
  - source: s3:main
    dest: backups
directory:
  - name: data
    source: "/tmp/data"
`,
			errSnippet: "at least one schedule timing",
		},
		{
			name: "missing drivers",
			content: `interval: 1m
repositories:
  - source: s3:main
    dest: backups
`,
			errSnippet: "at least one driver",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			specPath := filepath.Join(tmpDir, "backup.yml")
			if err := os.WriteFile(specPath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("failed to write spec: %v", err)
			}

			_, err := LoadSpec(specPath)
			if err == nil {
				t.Fatalf("expected error containing %q", tc.errSnippet)
			}
			if !strings.Contains(err.Error(), tc.errSnippet) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadSpecsFiltersFilesAndSetsFileName(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "data")
	_ = os.MkdirAll(sourceDir, 0755)

	validSpec := fmt.Sprintf(`interval: 1m
repositories:
  - source: s3:main
    dest: backups
directory:
  - name: data
    source: %q
`, filepath.ToSlash(sourceDir))

	if err := os.WriteFile(filepath.Join(tmpDir, "one.yml"), []byte(validSpec), 0644); err != nil {
		t.Fatalf("failed to write valid spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ignore.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("failed to write ignored file: %v", err)
	}

	defs, err := LoadSpecs(tmpDir)
	if err != nil {
		t.Fatalf("LoadSpecs failed: %v", err)
	}

	if len(defs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(defs))
	}
	if defs[0].FileName != "one.yml" {
		t.Fatalf("expected filename %q, got %q", "one.yml", defs[0].FileName)
	}
}
