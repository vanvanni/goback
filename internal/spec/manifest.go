package spec

import (
	"fmt"
	"time"

	"github.com/vanvanni/goback/internal/helper"
	"go.yaml.in/yaml/v4"
)

const CurrentManifestVersion = 1

type BackupManifest struct {
	ManifestVersion int                      `yaml:"manifest_version"`
	CreatedAt       time.Time                `yaml:"created_at"`
	SpecName        string                   `yaml:"spec_name,omitempty"`
	Encryption      BackupManifestEncryption `yaml:"encryption"`
	Repository      BackupManifestRepository `yaml:"repository,omitempty"`
	Repositories    []RepositorySpec         `yaml:"repositories,omitempty"`
	Items           []BackupItem             `yaml:"items"`
}

type BackupManifestEncryption struct {
	Enabled bool `yaml:"enabled"`
}

type BackupManifestRepository struct {
	Source string `yaml:"source,omitempty"`
	Dest   string `yaml:"dest,omitempty"`
}

type BackupItem struct {
	Type           string `yaml:"type"`
	Name           string `yaml:"name"`
	ArchiveName    string `yaml:"archive_name"`
	OriginalTarget string `yaml:"original_target,omitempty"`
	SourcePath     string `yaml:"source_path,omitempty"`
	Database       string `yaml:"database,omitempty"`
	VolumeName     string `yaml:"volume_name,omitempty"`
}

func LoadManifest(filePath string) (*BackupManifest, error) {
	data, err := helper.ReadFileSafe(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest BackupManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

func SaveManifest(filePath string, manifest *BackupManifest) error {
	if manifest == nil {
		return fmt.Errorf("manifest is required")
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := helper.WriteFileSafe(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}
