package spec

import (
	"os"
	"path/filepath"
)

var base string

func EnsureDirectoryStructure(basePath string) error {
	base = basePath
	if err := os.MkdirAll(GetSpecDirectory(), 0750); err != nil {
		return err
	}

	if _, err := os.Stat(GetCoreConfigPath()); os.IsNotExist(err) {
		if err := os.WriteFile(GetCoreConfigPath(), []byte{}, 0600); err != nil {
			return err
		}
	}

	return nil
}

func GetSpecDirectory() string {
	return filepath.Join(base, "specs.d")
}

func GetCoreConfigPath() string {
	return filepath.Join(base, "config.yml")
}
