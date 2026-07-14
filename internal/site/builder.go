package site

import (
	"html/template"
	"io/fs"
	"path/filepath"
)

type BuildOptions struct {
	ContentRoot string
	DestRoot    string
}

type Builder struct {
	contentRoot string
	destRoot    string
	config      SiteConfig
	theme       *template.Template
}

// Build compiles the site from the content directory: it loads pages, renders
// them concurrently, generates collection listings, and copies assets into the
// output directory
func Build(opts BuildOptions) error {
	b, err := newBuilder(opts)
	if err != nil {
		return err
	}

	paths, err := collectContent(b.contentRoot)
	if err != nil {
		return err
	}

	pages := make([]Page, 0, len(paths))
	for _, path := range paths {
		page, err := b.loadPage(path)
		if err != nil {
			return err
		}
		pages = append(pages, page)
	}

	// Change renderPages to a method on Builder
	if err := b.renderPages(pages); err != nil {
		return err
	}

	collections, err := groupCollections(pages, b.contentRoot)
	if err != nil {
		return err
	}

	for _, c := range collections {
		if c.Index != nil {
			continue // has index.md hence use that, renders via normal page path
		}
		// convert generateListingPage also to a method on builder
		if err := b.generateListingPage(c); err != nil {
			return err
		}
	}

	// convert copyAssets to a method on builder
	if err := b.copyAssets(); err != nil {
		return err
	}

	return nil
}

// newBuilder constructs a Builder from the given options, loading the site
// config and theme that the rest of the build depends on
func newBuilder(opts BuildOptions) (*Builder, error) {

	config, err := loadConfig(opts.ContentRoot)
	if err != nil {
		return nil, err
	}

	theme, err := loadTheme(filepath.Join("themes", config.Theme))
	if err != nil {
		return nil, err
	}

	// TODO: read output dir from buildOptions / site.toml instead of hardcoding "dist"
	return &Builder{
		contentRoot: opts.ContentRoot,
		destRoot:    opts.DestRoot,
		config:      config,
		theme:       theme,
	}, nil
}

// collectContent walks the content root recursively and returns the paths of
// all markdown (.md) files found in a slice
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
