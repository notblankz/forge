package site

import (
	"io/fs"
	"path/filepath"
)

type BuildOptions struct {
	ContentRoot string
}

func Build(opts BuildOptions) error {

	paths, err := collectContent(opts.ContentRoot)
	if err != nil {
		return err
	}

	pages := make([]Page, 0, len(paths))
	for _, path := range paths {
		page, err := loadPage(path, opts.ContentRoot)
		if err != nil {
			return err
		}
		pages = append(pages, page)
	}

	for _, page := range pages {
		html, err := page.render()
		if err != nil {
			return err
		}
		if err := page.write(html); err != nil {
			return err
		}
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
