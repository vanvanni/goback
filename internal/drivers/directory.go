package drivers

import (
	"fmt"
	"path/filepath"

	"github.com/vanvanni/goback/internal/helper"
)

type DirectoryDriver struct {
	Name   string `yaml:"name"`
	Source string `yaml:"source"`
}

func (d *DirectoryDriver) GetName() string {
	return d.Name
}

func (d *DirectoryDriver) Backup(workDir string) error {
	archivePath := filepath.Join(workDir, "dir-"+d.Name+".tar.gz")
	if err := helper.CompressDir(d.Source, archivePath); err != nil {
		return fmt.Errorf("failed to compress directory %s: %w", d.Name, err)
	}
	return nil
}
