package drivers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/vanvanni/goback/internal/engines"
	"github.com/vanvanni/goback/internal/helper"
)

type VolumeDriver struct {
	Name string `yaml:"name"`
}

func (d *VolumeDriver) GetName() string {
	return d.Name
}

func (d *VolumeDriver) Backup(ctx context.Context, dockerEngine *engines.Docker, workDir string) error {
	info, err := dockerEngine.GetVolumeByName(ctx, d.Name)
	if err != nil {
		return fmt.Errorf("failed to get volume info for %s: %w", d.Name, err)
	}

	// TODO: Pausing/Unpausing here

	archivePath := filepath.Join(workDir, "vol-"+d.Name+".tar.gz")
	if err := helper.CompressDir(info.MountPoint, archivePath); err != nil {
		return fmt.Errorf("failed to compress volume %s: %w", d.Name, err)
	}
	return nil
}
