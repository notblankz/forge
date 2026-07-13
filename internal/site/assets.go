package site

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// copyAssets mirrors the content assets directory into the output directory,
// copying every file. If the assets directory does not exist, it is a no-op
func copyAssets(contentRoot, destRoot string) error {
	assetsDir := filepath.Join(contentRoot, "assets")

	if _, err := os.Stat(assetsDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return filepath.WalkDir(assetsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(contentRoot, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destRoot, rel)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return copyFile(path, destPath)
	})

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
