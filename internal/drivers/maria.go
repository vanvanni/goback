package drivers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vanvanni/goback/internal/engines"
	"github.com/vanvanni/goback/internal/helper"
	"github.com/vanvanni/goback/internal/logging"
)

type MariaDriver struct {
	Name string `yaml:"name"`
}

func (d *MariaDriver) GetName() string {
	return d.Name
}

func (d *MariaDriver) Backup(ctx context.Context, dumpEngine *engines.Dump, mariaConfig engines.MariaDBDumpConfig, workDir string) error {
	dumpFile := filepath.Join(workDir, d.Name+".sql")

	host := mariaConfig.Host
	if host == "" {
		host = "localhost"
	}

	port := mariaConfig.Port
	if port == "" {
		port = "3306"
	}

	user := mariaConfig.User
	if user == "" {
		user = "root"
	}

	cfg := engines.MariaDBDumpConfig{
		Host:       host,
		Port:       port,
		User:       user,
		Password:   mariaConfig.Password,
		Database:   d.Name,
		OutputFile: dumpFile,
	}

	cmd, err := dumpEngine.DumpMariaDB(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to prepare dump for %s: %w", d.Name, err)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute dump for %s: %w", d.Name, err)
	}

	archivePath := filepath.Join(workDir, "mariadb-"+d.Name+".tar.gz")
	if err := helper.CompressFile(dumpFile, archivePath); err != nil {
		return fmt.Errorf("failed to compress dump for %s: %w", d.Name, err)
	}

	if err := os.Remove(dumpFile); err != nil {
		logging.Log.Warn().Err(err).Str("file", dumpFile).Msg("Failed to remove temp dump file")
	}
	return nil
}
