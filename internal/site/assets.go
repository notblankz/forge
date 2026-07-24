package site

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// copyTree recursively copies every file under srcDir into destDir, preserving
// each file's path relative to relBase. A no-op if srcDir doesn't exist, since
// both content/assets/ and a theme's static/ are optional
// TODO: still need to remove orphaned assets from the destDir
func copyTree(srcDir, relBase, destDir string) error {
	if _, err := os.Stat(srcDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(relBase, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, rel)

		// Skip if the destination is already up to date with the source
		srcInfo, err := d.Info()
		if err != nil {
			return err
		}
		// Here we check if the files at src and dest are 1) same size and 2) dest is not older than src
		if destInfo, err := os.Stat(destPath); err == nil &&
			destInfo.Size() == srcInfo.Size() &&
			!srcInfo.ModTime().After(destInfo.ModTime()) {
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return copyFile(path, destPath)
	})
}

// copyAssets mirrors content/assets/ into <dest>/assets/
func (b *Builder) copyAssets() error {
	assetsDir := filepath.Join(b.contentDir, "assets")
	return copyTree(assetsDir, b.contentDir, b.destDir)
}

// copyThemeAssets mirrors the active theme's static/ directory into dest
func (b *Builder) copyThemeAssets() error {
	staticDir := filepath.Join(b.themeDir, "static")
	return copyTree(staticDir, b.themeDir, b.destDir)
}

// copyFile streams the contents of src into a newly created dest file.
func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	// Ensure buffered data is flushed to disk before returning
	return out.Close()
}
