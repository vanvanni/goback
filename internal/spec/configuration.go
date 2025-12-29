package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-co-op/gocron/v2"
	"github.com/vanvanni/goback/internal/drivers"
	"github.com/vanvanni/goback/internal/engines"
	"github.com/vanvanni/goback/internal/helper"
	"go.yaml.in/yaml/v4"
)

type CoreConfig struct {
	S3      map[string]engines.S3Config `yaml:"s3"`
	MariaDB engines.MariaDBDumpConfig   `yaml:"mariadb"`
}

type BackupSpec struct {
	Crontab      string           `yaml:"crontab"`
	Duration     string           `yaml:"duration"`
	Repositories []RepositorySpec `yaml:"repositories"`
	Retention    RetentionPolicy  `yaml:"retention"`

	// Drivers
	MariaDB   []drivers.MariaDriver     `yaml:"mariadb"`
	Directory []drivers.DirectoryDriver `yaml:"directory"`
	Volumes   []drivers.VolumeDriver    `yaml:"volumes"`
}

type RepositorySpec struct {
	Source string `yaml:"source"`
	Dest   string `yaml:"dest"`
}

type RetentionPolicy struct {
	KeepLast int `yaml:"keep_last"`
}

type BackupDefinition struct {
	FileName      string
	Repositories  []RepositorySpec
	Retention     RetentionPolicy
	ScheduleTimer gocron.JobDefinition
	MariaDB       []drivers.MariaDriver
	Directory     []drivers.DirectoryDriver
	Volumes       []drivers.VolumeDriver
}

func LoadConfig(filePath string) (*CoreConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read core config file: %w", err)
	}

	var coreConfig CoreConfig
	if err := yaml.Unmarshal(data, &coreConfig); err != nil {
		return nil, fmt.Errorf("failed to parse core config file: %w", err)
	}

	return &coreConfig, nil
}

func LoadSpec(filePath string) (*BackupDefinition, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	var backupSpec BackupSpec
	if err := yaml.Unmarshal(data, &backupSpec); err != nil {
		return nil, fmt.Errorf("failed to parse spec file: %w", err)
	}

	if len(backupSpec.Repositories) == 0 {
		return nil, fmt.Errorf("at least one repository is required")
	}

	if backupSpec.Crontab == "" && backupSpec.Duration == "" {
		return nil, fmt.Errorf("at least one schedule timing (crontab or duration) is required")
	}

	if backupSpec.MariaDB == nil && backupSpec.Directory == nil && backupSpec.Volumes == nil {
		return nil, fmt.Errorf("at least one driver (mariadb, directory, or volumes) is required")
	}

	var scheduleTimer gocron.JobDefinition

	if backupSpec.Duration != "" {
		d, err := helper.ParseDuration(backupSpec.Duration)
		if err != nil {
			return nil, fmt.Errorf("failed parsing duration")
		}

		scheduleTimer = gocron.DurationJob(d)
	}

	if backupSpec.Crontab != "" {
		scheduleTimer = gocron.CronJob(backupSpec.Crontab, false)
	}

	definition := &BackupDefinition{
		Repositories:  backupSpec.Repositories,
		Retention:     backupSpec.Retention,
		ScheduleTimer: scheduleTimer,
		MariaDB:       backupSpec.MariaDB,
		Directory:     backupSpec.Directory,
		Volumes:       backupSpec.Volumes,
	}
	return definition, nil
}

func LoadSpecs(dirPath string) ([]*BackupDefinition, error) {
	var defs []*BackupDefinition

	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read specs directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if !strings.HasSuffix(file.Name(), ".yml") && !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}

		filePath := filepath.Join(dirPath, file.Name())
		def, err := LoadSpec(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read spec file %s: %w", filePath, err)
		}

		def.FileName = file.Name()
		defs = append(defs, def)
	}

	return defs, nil
}
