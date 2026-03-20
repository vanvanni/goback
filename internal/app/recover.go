package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vanvanni/goback/internal/engines"
	"github.com/vanvanni/goback/internal/helper"
	"github.com/vanvanni/goback/internal/spec"
)

type RecoveryDownloader interface {
	DownloadFile(ctx context.Context, remoteKey, localPath string) error
}

type RecoveryRuntime struct {
	Repositories map[string]*engines.S3
}

type RecoveryTask struct {
	RemoteKey     string
	OutputPath    string
	DecryptionKey string
	Downloader    RecoveryDownloader
}

func CreateRecoveryRuntime(ctx context.Context, conf *spec.CoreConfig) (*RecoveryRuntime, error) {
	repositories := make(map[string]*engines.S3)
	for name, s3cfg := range conf.S3 {
		client, err := engines.NewClient(ctx, s3cfg)
		if err != nil {
			return nil, fmt.Errorf("could not create S3 connection %q: %w", name, err)
		}
		repositories[name] = client
	}

	return &RecoveryRuntime{Repositories: repositories}, nil
}

func (rt *RecoveryTask) Run(ctx context.Context) (*spec.BackupManifest, error) {
	if rt.Downloader == nil {
		return nil, fmt.Errorf("downloader is required")
	}
	if strings.TrimSpace(rt.RemoteKey) == "" {
		return nil, fmt.Errorf("remote key is required")
	}
	if err := prepareRecoveryOutput(rt.OutputPath); err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "recover-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	localArchive := filepath.Join(tempDir, helper.SafeFilename(filepath.Base(rt.RemoteKey)))
	if err := rt.Downloader.DownloadFile(ctx, rt.RemoteKey, localArchive); err != nil {
		return nil, fmt.Errorf("failed to download backup: %w", err)
	}

	outerArchive, err := rt.resolveOuterArchive(localArchive)
	if err != nil {
		return nil, err
	}

	extractDir := filepath.Join(tempDir, "bundle")
	if err := os.MkdirAll(extractDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create extraction dir: %w", err)
	}
	if err := helper.ExtractTarGz(outerArchive, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract backup archive: %w", err)
	}

	bundleRoot, err := resolveBundleRoot(extractDir)
	if err != nil {
		return nil, err
	}

	manifest, err := resolveManifest(bundleRoot)
	if err != nil {
		return nil, err
	}

	if err := extractRecoveryItems(bundleRoot, rt.OutputPath, manifest); err != nil {
		return nil, err
	}

	if err := spec.SaveManifest(filepath.Join(rt.OutputPath, "manifest.yml"), manifest); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	return manifest, nil
}

func (rt *RecoveryTask) resolveOuterArchive(localArchive string) (string, error) {
	if !strings.HasSuffix(localArchive, ".enc") {
		return localArchive, nil
	}
	if strings.TrimSpace(rt.DecryptionKey) == "" {
		return "", fmt.Errorf("decryption key is required for encrypted backups")
	}

	decrypted := strings.TrimSuffix(localArchive, ".enc")
	if err := helper.DecryptFile(localArchive, decrypted, rt.DecryptionKey); err != nil {
		return "", fmt.Errorf("failed to decrypt backup: %w", err)
	}
	return decrypted, nil
}

func resolveBundleRoot(extractDir string) (string, error) {
	manifestPath := filepath.Join(extractDir, "manifest.yml")
	if _, err := os.Stat(manifestPath); err == nil {
		return extractDir, nil
	}

	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", fmt.Errorf("failed to inspect extracted bundle: %w", err)
	}

	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(extractDir, entries[0].Name()), nil
	}

	return extractDir, nil
}

func resolveManifest(bundleRoot string) (*spec.BackupManifest, error) {
	manifestPath := filepath.Join(bundleRoot, "manifest.yml")
	if _, err := os.Stat(manifestPath); err == nil {
		return spec.LoadManifest(manifestPath)
	}

	items, err := inferManifestItems(bundleRoot)
	if err != nil {
		return nil, err
	}

	return &spec.BackupManifest{
		ManifestVersion: 0,
		Items:           items,
	}, nil
}

func inferManifestItems(bundleRoot string) ([]spec.BackupItem, error) {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect backup bundle: %w", err)
	}

	items := make([]spec.BackupItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		switch {
		case strings.HasPrefix(name, "dir-") && strings.HasSuffix(name, ".tar.gz"):
			itemName := strings.TrimSuffix(strings.TrimPrefix(name, "dir-"), ".tar.gz")
			items = append(items, spec.BackupItem{Type: "directory", Name: itemName, ArchiveName: name})
		case strings.HasPrefix(name, "mariadb-") && strings.HasSuffix(name, ".tar.gz"):
			itemName := strings.TrimSuffix(strings.TrimPrefix(name, "mariadb-"), ".tar.gz")
			items = append(items, spec.BackupItem{Type: "mariadb", Name: itemName, ArchiveName: name})
		case strings.HasPrefix(name, "vol-") && strings.HasSuffix(name, ".tar.gz"):
			itemName := strings.TrimSuffix(strings.TrimPrefix(name, "vol-"), ".tar.gz")
			items = append(items, spec.BackupItem{Type: "volume", Name: itemName, ArchiveName: name})
		}
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no recoverable items found in backup")
	}

	return items, nil
}

func extractRecoveryItems(bundleRoot, outputPath string, manifest *spec.BackupManifest) error {
	if manifest == nil {
		return fmt.Errorf("manifest is required")
	}

	for _, item := range manifest.Items {
		archivePath := filepath.Join(bundleRoot, item.ArchiveName)
		if _, err := os.Stat(archivePath); err != nil {
			return fmt.Errorf("backup item archive %q was not found", item.ArchiveName)
		}

		switch item.Type {
		case "directory":
			targetPath := filepath.Join(outputPath, "directory", item.Name)
			if err := helper.ExtractTarGz(archivePath, targetPath); err != nil {
				return fmt.Errorf("failed to extract directory %q: %w", item.Name, err)
			}
		case "mariadb":
			targetPath := filepath.Join(outputPath, "mariadb", item.Name+".sql")
			if err := helper.ExtractGzipFile(archivePath, targetPath); err != nil {
				return fmt.Errorf("failed to extract mariadb dump %q: %w", item.Name, err)
			}
		case "volume":
			targetPath := filepath.Join(outputPath, "volumes", item.Name)
			if err := helper.ExtractTarGz(archivePath, targetPath); err != nil {
				return fmt.Errorf("failed to extract volume %q: %w", item.Name, err)
			}
		default:
			return fmt.Errorf("unsupported backup item type %q", item.Type)
		}
	}

	return nil
}

func prepareRecoveryOutput(outputPath string) error {
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("output path is required")
	}

	info, err := os.Stat(outputPath)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("output path %q is not a directory", outputPath)
		}

		entries, err := os.ReadDir(outputPath)
		if err != nil {
			return fmt.Errorf("failed to inspect output path %q: %w", outputPath, err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("output path %q must be empty", outputPath)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect output path %q: %w", outputPath, err)
	}

	if err := os.MkdirAll(outputPath, 0750); err != nil {
		return fmt.Errorf("failed to create output path %q: %w", outputPath, err)
	}

	return nil
}
