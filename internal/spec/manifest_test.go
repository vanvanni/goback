package spec

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManifestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yml")

	want := &BackupManifest{
		ManifestVersion: CurrentManifestVersion,
		CreatedAt:       time.Unix(123, 0).UTC(),
		SpecName:        "daily.yml",
		Encryption: BackupManifestEncryption{
			Enabled: true,
		},
		Repository: BackupManifestRepository{
			Source: "s3:main",
			Dest:   "backups",
		},
		Repositories: []RepositorySpec{
			{Source: "s3:main", Dest: "backups"},
		},
		Items: []BackupItem{
			{
				Type:           "directory",
				Name:           "data",
				ArchiveName:    "dir-data.tar.gz",
				OriginalTarget: "/srv/data",
				SourcePath:     "/srv/data",
			},
		},
	}

	if err := SaveManifest(path, want); err != nil {
		t.Fatalf("SaveManifest failed: %v", err)
	}

	got, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if got.ManifestVersion != want.ManifestVersion {
		t.Fatalf("expected version %d, got %d", want.ManifestVersion, got.ManifestVersion)
	}
	if got.SpecName != want.SpecName {
		t.Fatalf("expected spec %q, got %q", want.SpecName, got.SpecName)
	}
	if got.Repository.Source != want.Repository.Source || got.Repository.Dest != want.Repository.Dest {
		t.Fatalf("unexpected repository: %#v", got.Repository)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got.Items))
	}
	if got.Items[0].ArchiveName != "dir-data.tar.gz" {
		t.Fatalf("unexpected archive name %q", got.Items[0].ArchiveName)
	}
}
