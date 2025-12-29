package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-co-op/gocron/v2"
	"github.com/vanvanni/goback/internal/drivers"
	"github.com/vanvanni/goback/internal/helper"
	"go.yaml.in/yaml/v4"
)

type S3Config struct {
	Bucket         string `toml:"bucket" yaml:"bucket"`
	Region         string `toml:"region" yaml:"region"`
	AccessKey      string `toml:"access_key" yaml:"access_key"`
	SecretKey      string `toml:"secret_key" yaml:"secret_key"`
	Endpoint       string `toml:"endpoint" yaml:"endpoint"`
	ForcePathStyle bool   `toml:"force-path" yaml:"force-path"`
}

type CoreConfig struct {
	S3 map[string]S3Config `yaml:"s3"`
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

	var spec BackupSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse spec file: %w", err)
	}

	if len(spec.Repositories) == 0 {
		return nil, fmt.Errorf("at least one repository is required")
	}

	if spec.Crontab == "" && spec.Duration == "" {
		return nil, fmt.Errorf("at least one schedule timing (crontab or duration) is required")
	}

	if spec.MariaDB == nil && spec.Directory == nil && spec.Volumes == nil {
		return nil, fmt.Errorf("at least one driver (mariadb, directory, or volumes) is required")
	}

	var scheduleTimer gocron.JobDefinition

	if spec.Duration != "" {
		d, err := helper.ParseDuration(spec.Duration)
		if err != nil {
			return nil, fmt.Errorf("failed parsing duration")
		}

		scheduleTimer = gocron.DurationJob(d)
	}

	if spec.Crontab != "" {
		scheduleTimer = gocron.CronJob(spec.Crontab, false)
	}

	definition := &BackupDefinition{
		Repositories:  spec.Repositories,
		Retention:     spec.Retention,
		ScheduleTimer: scheduleTimer,
		MariaDB:       spec.MariaDB,
		Directory:     spec.Directory,
		Volumes:       spec.Volumes,
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
