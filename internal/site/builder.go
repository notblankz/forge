package site

import (
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

	pages := make([]Page, 0, len(paths))
	for _, path := range paths {
		page, err := loadPage(path)
		if err != nil {
			return err
		}
		pages = append(pages, page)
	}

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
