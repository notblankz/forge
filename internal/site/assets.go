package site

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/notblankz/forge/internal/engine"
)

// @assets - node which mirrors content/assets to dest/assets
type assetsNode struct {
	contentDir string
	destDir    string
}

func (a assetsNode) Hash(string) (string, error) {
	h, err := engine.HashDir(filepath.Join(a.contentDir, "assets"))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return h, nil
}

func (a assetsNode) Build(*engine.BuildCtx, string) (engine.Result, error) {
	dests, err := copyTree(filepath.Join(a.contentDir, "assets"), a.contentDir, a.destDir)
	return engine.Result{
		Outputs: dests,
	}, err
}

// @theme-static - node that mirrors theme's static into dist/
type themeStaticNode struct {
	themeDir string
	destDir  string
}

func (t themeStaticNode) Hash(string) (string, error) {
	h, err := engine.HashDir(filepath.Join(t.themeDir, "static"))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return h, err
}

func (t themeStaticNode) Build(*engine.BuildCtx, string) (engine.Result, error) {
	dests, err := copyTree(filepath.Join(t.themeDir, "static"), t.themeDir, t.destDir)
	return engine.Result{
		Outputs: dests,
	}, err
}

// copyTree recursively copies every file under srcDir into destDir, preserving
// each file's path relative to relBase. A no-op if srcDir doesn't exist, since
// both content/assets/ and a theme's static/ are optional
func copyTree(srcDir, relBase, destDir string) ([]string, error) {
	if _, err := os.Stat(srcDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var dests []string
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
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
		dests = append(dests, destPath)

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

	if err != nil {
		return nil, err
	}

	return dests, nil
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
