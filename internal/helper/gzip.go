package helper

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func CompressDir(dir string, dest string) (err error) {
	out, err := OpenWriteOnlyFile(dest, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()

	return fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(path)

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			f, err := root.Open(path)
			if err != nil {
				return err
			}

			if _, err := io.Copy(tw, f); err != nil {
				_ = f.Close()
				return err
			}

			if err := f.Close(); err != nil {
				return err
			}
		}
		return nil
	})
}

func CompressFile(src string, dest string) error {
	in, err := OpenReadOnlyFile(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := OpenWriteOnlyFile(dest, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	gz := gzip.NewWriter(out)
	defer gz.Close()

	_, err = io.Copy(gz, in)
	return err
}
