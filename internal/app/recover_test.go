package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vanvanni/goback/internal/helper"
	"github.com/vanvanni/goback/internal/spec"
)

func TestRecoveryTaskRunExtractsDirectoryBundle(t *testing.T) {
	bundle := createDirectoryBundle(t, true)
	outputPath := filepath.Join(t.TempDir(), "restore")

	task := &RecoveryTask{
		RemoteKey:  "backups/daily.tar.gz",
		OutputPath: outputPath,
		Downloader: fakeDownloader{source: bundle},
	}

	manifest, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(manifest.Items) != 1 {
		t.Fatalf("expected 1 manifest item, got %d", len(manifest.Items))
	}

	data, err := os.ReadFile(filepath.Join(outputPath, "directory", "data", "hello.txt"))
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}
	if string(data) != "hello recover" {
		t.Fatalf("expected restored content %q, got %q", "hello recover", string(data))
	}

	writtenManifest, err := spec.LoadManifest(filepath.Join(outputPath, "manifest.yml"))
	if err != nil {
		t.Fatalf("failed to load written manifest: %v", err)
	}
	if writtenManifest.Items[0].ArchiveName != "dir-data.tar.gz" {
		t.Fatalf("unexpected archive name %q", writtenManifest.Items[0].ArchiveName)
	}
}

func TestRecoveryTaskRunExtractsEncryptedBundleWithKey(t *testing.T) {
	plainBundle := createDirectoryBundle(t, true)
	encryptedBundle := plainBundle + ".enc"
	if err := helper.EncryptFile(plainBundle, encryptedBundle, "secret"); err != nil {
		t.Fatalf("failed to encrypt bundle: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "restore")
	task := &RecoveryTask{
		RemoteKey:     "backups/daily.tar.gz.enc",
		OutputPath:    outputPath,
		DecryptionKey: "secret",
		Downloader:    fakeDownloader{source: encryptedBundle},
	}

	if _, err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputPath, "directory", "data", "hello.txt"))
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}
	if string(data) != "hello recover" {
		t.Fatalf("expected restored content %q, got %q", "hello recover", string(data))
	}
}

func TestRecoveryTaskRunInfersManifestForLegacyBundle(t *testing.T) {
	bundle := createDirectoryBundle(t, false)
	outputPath := filepath.Join(t.TempDir(), "restore")

	task := &RecoveryTask{
		RemoteKey:  "backups/legacy.tar.gz",
		OutputPath: outputPath,
		Downloader: fakeDownloader{source: bundle},
	}

	manifest, err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if manifest.ManifestVersion != 0 {
		t.Fatalf("expected inferred manifest version 0, got %d", manifest.ManifestVersion)
	}

	writtenManifest, err := spec.LoadManifest(filepath.Join(outputPath, "manifest.yml"))
	if err != nil {
		t.Fatalf("failed to load written manifest: %v", err)
	}
	if len(writtenManifest.Items) != 1 {
		t.Fatalf("expected 1 inferred item, got %d", len(writtenManifest.Items))
	}
	if writtenManifest.Items[0].Name != "data" {
		t.Fatalf("unexpected inferred item name %q", writtenManifest.Items[0].Name)
	}
}

func TestRecoveryTaskRunExtractsMixedBundle(t *testing.T) {
	bundle := createMixedBundle(t)
	outputPath := filepath.Join(t.TempDir(), "restore")

	task := &RecoveryTask{
		RemoteKey:  "backups/mixed.tar.gz",
		OutputPath: outputPath,
		Downloader: fakeDownloader{source: bundle},
	}

	if _, err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	dirData, err := os.ReadFile(filepath.Join(outputPath, "directory", "data", "hello.txt"))
	if err != nil {
		t.Fatalf("failed to read directory payload: %v", err)
	}
	if string(dirData) != "hello recover" {
		t.Fatalf("unexpected directory payload %q", string(dirData))
	}

	sqlData, err := os.ReadFile(filepath.Join(outputPath, "mariadb", "app.sql"))
	if err != nil {
		t.Fatalf("failed to read mariadb dump: %v", err)
	}
	if string(sqlData) != "select 1;" {
		t.Fatalf("unexpected mariadb dump %q", string(sqlData))
	}

	volumeData, err := os.ReadFile(filepath.Join(outputPath, "volumes", "shared", "hello.txt"))
	if err != nil {
		t.Fatalf("failed to read volume payload: %v", err)
	}
	if string(volumeData) != "volume" {
		t.Fatalf("unexpected volume payload %q", string(volumeData))
	}
}

func TestRecoveryTaskRunRequiresEmptyOutputDir(t *testing.T) {
	bundle := createDirectoryBundle(t, true)
	outputPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(outputPath, "existing.txt"), []byte("keep"), 0644); err != nil {
		t.Fatalf("failed to seed output path: %v", err)
	}

	task := &RecoveryTask{
		RemoteKey:  "backups/daily.tar.gz",
		OutputPath: outputPath,
		Downloader: fakeDownloader{source: bundle},
	}

	_, err := task.Run(context.Background())
	if err == nil {
		t.Fatal("expected output path validation error")
	}
	if !strings.Contains(err.Error(), "must be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecoveryTaskRunRequiresKeyForEncryptedBackup(t *testing.T) {
	plainBundle := createDirectoryBundle(t, true)
	encryptedBundle := plainBundle + ".enc"
	if err := helper.EncryptFile(plainBundle, encryptedBundle, "secret"); err != nil {
		t.Fatalf("failed to encrypt bundle: %v", err)
	}

	task := &RecoveryTask{
		RemoteKey:  "backups/daily.tar.gz.enc",
		OutputPath: filepath.Join(t.TempDir(), "restore"),
		Downloader: fakeDownloader{source: encryptedBundle},
	}

	_, err := task.Run(context.Background())
	if err == nil {
		t.Fatal("expected missing key error")
	}
	if !strings.Contains(err.Error(), "decryption key is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakeDownloader struct {
	source string
}

func (f fakeDownloader) DownloadFile(_ context.Context, _ string, localPath string) error {
	data, err := os.ReadFile(f.source)
	if err != nil {
		return err
	}
	return os.WriteFile(localPath, data, 0600)
}

func createDirectoryBundle(t *testing.T, includeManifest bool) string {
	t.Helper()

	manifest := spec.BackupManifest{
		ManifestVersion: spec.CurrentManifestVersion,
		SpecName:        "daily.yml",
		Items: []spec.BackupItem{{
			Type:           "directory",
			Name:           "data",
			ArchiveName:    "dir-data.tar.gz",
			OriginalTarget: "/srv/data",
			SourcePath:     "/srv/data",
		}},
	}

	workDir := t.TempDir()
	contentDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(contentDir, "hello.txt"), []byte("hello recover"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}
	if err := helper.CompressDir(contentDir, filepath.Join(workDir, "dir-data.tar.gz")); err != nil {
		t.Fatalf("failed to create inner archive: %v", err)
	}
	if includeManifest {
		if err := spec.SaveManifest(filepath.Join(workDir, "manifest.yml"), &manifest); err != nil {
			t.Fatalf("failed to save manifest: %v", err)
		}
	}

	return compressWorkDir(t, workDir)
}

func createMixedBundle(t *testing.T) string {
	t.Helper()

	workDir := t.TempDir()

	dirContent := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirContent, "hello.txt"), []byte("hello recover"), 0644); err != nil {
		t.Fatalf("failed to write directory payload: %v", err)
	}
	if err := helper.CompressDir(dirContent, filepath.Join(workDir, "dir-data.tar.gz")); err != nil {
		t.Fatalf("failed to create directory archive: %v", err)
	}

	sqlFile := filepath.Join(t.TempDir(), "dump.sql")
	if err := os.WriteFile(sqlFile, []byte("select 1;"), 0644); err != nil {
		t.Fatalf("failed to write SQL file: %v", err)
	}
	if err := helper.CompressFile(sqlFile, filepath.Join(workDir, "mariadb-app.tar.gz")); err != nil {
		t.Fatalf("failed to create mariadb archive: %v", err)
	}

	volumeContent := t.TempDir()
	if err := os.WriteFile(filepath.Join(volumeContent, "hello.txt"), []byte("volume"), 0644); err != nil {
		t.Fatalf("failed to write volume payload: %v", err)
	}
	if err := helper.CompressDir(volumeContent, filepath.Join(workDir, "vol-shared.tar.gz")); err != nil {
		t.Fatalf("failed to create volume archive: %v", err)
	}

	manifest := spec.BackupManifest{
		ManifestVersion: spec.CurrentManifestVersion,
		SpecName:        "mixed.yml",
		Items: []spec.BackupItem{
			{Type: "directory", Name: "data", ArchiveName: "dir-data.tar.gz"},
			{Type: "mariadb", Name: "app", ArchiveName: "mariadb-app.tar.gz"},
			{Type: "volume", Name: "shared", ArchiveName: "vol-shared.tar.gz"},
		},
	}
	if err := spec.SaveManifest(filepath.Join(workDir, "manifest.yml"), &manifest); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	return compressWorkDir(t, workDir)
}

func compressWorkDir(t *testing.T, workDir string) string {
	t.Helper()

	outer := filepath.Join(t.TempDir(), "bundle.tar.gz")
	if err := helper.CompressDir(workDir, outer); err != nil {
		t.Fatalf("failed to create outer archive: %v", err)
	}
	return outer
}
