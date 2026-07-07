package site

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

type BuildOptions struct {
	Root string
}

func Build(opts BuildOptions) error {

	paths, err := collectContent(opts.Root)
	if err != nil {
		return err
	}
	fmt.Println(paths)

	return nil
}

func collectContent(root string) ([]string, error) {

	res := make([]string, 0)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {

		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(d.Name())
		if ext == ".md" {
			res = append(res, path)
		}

		return nil
	})

	return res, err
}
