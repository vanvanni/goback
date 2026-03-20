package helper

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ExtractTarGz(src, dest string) error {
	in, err := OpenReadOnlyFile(src)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer in.Close()

	gz, err := gzip.NewReader(in)
	if err != nil {
		return fmt.Errorf("failed to read gzip archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		targetPath, err := safeExtractPath(dest, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0750); err != nil {
				return fmt.Errorf("failed to create directory %q: %w", targetPath, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0750); err != nil {
				return fmt.Errorf("failed to create parent directory for %q: %w", targetPath, err)
			}
			out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
			if err != nil {
				return fmt.Errorf("failed to create file %q: %w", targetPath, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return fmt.Errorf("failed to extract file %q: %w", targetPath, err)
			}
			if err := out.Close(); err != nil {
				return fmt.Errorf("failed to close file %q: %w", targetPath, err)
			}
		default:
			return fmt.Errorf("unsupported archive entry type %d for %q", hdr.Typeflag, hdr.Name)
		}
	}
}

func ExtractGzipFile(src, dest string) error {
	in, err := OpenReadOnlyFile(src)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer in.Close()

	gz, err := gzip.NewReader(in)
	if err != nil {
		return fmt.Errorf("failed to read gzip archive: %w", err)
	}
	defer gz.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0750); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	out, err := OpenWriteOnlyFile(dest, 0600)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, gz); err != nil {
		return fmt.Errorf("failed to extract gzip file: %w", err)
	}

	return nil
}

func RemoveContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %q: %w", dir, err)
	}

	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("failed to remove %q: %w", entry.Name(), err)
		}
	}

	return nil
}

func safeExtractPath(root, name string) (string, error) {
	cleanName := filepath.Clean(filepath.FromSlash(name))
	if cleanName == "." {
		return root, nil
	}
	if filepath.IsAbs(cleanName) {
		return "", fmt.Errorf("archive entry %q must not be absolute", name)
	}

	targetPath := filepath.Join(root, cleanName)
	relative, err := filepath.Rel(root, targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve archive entry %q: %w", name, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q escapes target directory", name)
	}

	return targetPath, nil
}
