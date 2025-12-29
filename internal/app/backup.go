package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vanvanni/goback/internal/drivers"
	"github.com/vanvanni/goback/internal/engines"
	"github.com/vanvanni/goback/internal/logging"
	"github.com/vanvanni/goback/internal/spec"
	"github.com/walle/targz"
)

type BackupTask struct {
	Definition    *spec.BackupDefinition
	MariaDB       []drivers.MariaDriver
	Directory     []drivers.DirectoryDriver
	Volumes       []drivers.VolumeDriver
	Repositories  []spec.RepositorySpec
	Retention     spec.RetentionPolicy
	MariaDBConfig engines.MariaDBDumpConfig

	// Engines
	docker  *engines.Docker
	dump    *engines.Dump
	engines map[string]engines.Engine
}

func (bt *BackupTask) Docker() *engines.Docker {
	return bt.docker
}

func (bt *BackupTask) Dump() *engines.Dump {
	return bt.dump
}

func (bt *BackupTask) GetEngine(key string) engines.Engine {
	return bt.engines[key]
}

func (bt *BackupTask) GetS3(name string) *engines.S3 {
	engine := bt.engines["s3:"+name]
	if engine == nil {
		return nil
	}
	s3Engine, ok := engine.(*engines.S3)
	if !ok {
		return nil
	}
	return s3Engine
}

func (bt *BackupTask) Run(ctx context.Context) (string, error) {
	workDir, err := os.MkdirTemp("", "backup-"+bt.Definition.FileName)
	logging.Log.Info().Str("backup-dir", workDir).Msg("Created backup directory")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	// defer os.RemoveAll(workDir)

	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	wg.Add(3)

	go func() {
		defer wg.Done()
		for _, db := range bt.MariaDB {
			if err := db.Backup(ctx, bt.dump, bt.MariaDBConfig, workDir); err != nil {
				errChan <- fmt.Errorf("failed to process mariadb backups: %w", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for _, dir := range bt.Directory {
			if err := dir.Backup(workDir); err != nil {
				errChan <- fmt.Errorf("failed to process directory backups: %w", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		if bt.docker == nil {
			if len(bt.Volumes) > 0 {
				logging.Log.Warn().Msg("Docker engine not available, skipping volume backups")
			}
			return
		}

		for _, vol := range bt.Volumes {
			if err := vol.Backup(ctx, bt.docker, workDir); err != nil {
				errChan <- fmt.Errorf("failed to process volume backups: %w", err)
				return
			}
		}
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return "", err
		}
	}

	finalArchive := filepath.Join(os.TempDir(), fmt.Sprintf("backup-%s-%d.tar.gz",
		bt.Definition.FileName, time.Now().Unix()))

	if err := targz.Compress(workDir, finalArchive); err != nil {
		return "", fmt.Errorf("failed to create final archive: %w", err)
	}

	for _, repo := range bt.Repositories {
		parts := strings.Split(repo.Source, ":")
		if len(parts) != 2 {
			logging.Log.Warn().Str("source", repo.Source).Msg("Invalid repository source format")
			continue
		}

		if parts[0] == "s3" {
			engine := bt.GetS3(parts[1])
			if engine == nil {
				logging.Log.Error().Str("repository", repo.Source).Msg("S3 engine not found for repository")
				continue
			}

			logging.Log.Info().Str("repository", repo.Source).Msg("Uploading archive")
			if err := engine.UploadFile(ctx, finalArchive, repo.Dest); err != nil {
				return "", fmt.Errorf("failed to upload to %s: %w", repo.Source, err)
			}

			if bt.Retention.KeepLast > 0 {
				logging.Log.Info().Str("repository", repo.Source).Int("keep", bt.Retention.KeepLast).Msg("Enforcing retention policy")
				if err := engine.KeepMax(ctx, repo.Dest, bt.Retention.KeepLast); err != nil {
					logging.Log.Error().Err(err).Str("repository", repo.Source).Msg("Failed to enforce retention policy")
				}
			}
		}
	}

	return finalArchive, nil
}
