package helper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/safeopen"
)

func ReadFileSafe(path string) ([]byte, error) {
	dir, name, err := splitPath(path)
	if err != nil {
		return nil, err
	}

	data, err := safeopen.ReadFileBeneath(dir, name)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func WriteFileSafe(path string, data []byte, perm os.FileMode) error {
	dir, name, err := splitPath(path)
	if err != nil {
		return err
	}

	return safeopen.WriteFileBeneath(dir, name, data, perm)
}

func OpenReadOnlyFile(path string) (*os.File, error) {
	dir, name, err := splitPath(path)
	if err != nil {
		return nil, err
	}

	file, err := safeopen.OpenBeneath(dir, name)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func OpenWriteOnlyFile(path string, perm os.FileMode) (*os.File, error) {
	dir, name, err := splitPath(path)
	if err != nil {
		return nil, err
	}

	file, err := safeopen.OpenFileBeneath(dir, name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func splitPath(path string) (string, string, error) {
	if strings.TrimSpace(path) == "" {
		return "", "", fmt.Errorf("path cannot be empty")
	}

	clean := filepath.Clean(path)
	name := filepath.Base(clean)
	if name == "." || name == string(filepath.Separator) {
		return "", "", fmt.Errorf("path %q must reference a file", path)
	}

	dir := filepath.Dir(clean)
	if dir == "" {
		dir = "."
	}

	return dir, name, nil
}
