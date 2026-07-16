package site

import (
	"html/template"
	"io/fs"
	"path/filepath"
)

type BuildOptions struct {
	SiteRoot string
	DestDir  string
}

type Builder struct {
	contentDir string
	themeDir   string
	destDir    string
	config     SiteConfig
	theme      *template.Template
}

// Build compiles the site from the content directory: it loads pages, renders
// them concurrently, generates collection listings, and copies assets into the
// output directory
func Build(opts BuildOptions) error {
	b, err := newBuilder(opts)
	if err != nil {
		return err
	}

	paths, err := collectContent(b.contentDir)
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

	// Render all the standalone pages
	if err := b.renderPages(pages); err != nil {
		return err
	}

	// Create the collections map for the different collections in content/
	collections, err := groupCollections(pages, b.contentDir)
	if err != nil {
		return err
	}

	for _, c := range collections {
		if c.Index != nil {
			continue // has index.md hence use that, renders via normal page path
		}
		if err := b.generateListingPage(c); err != nil {
			return err
		}
	}

	if err := b.copyAssets(); err != nil {
		return err
	}

	if err := b.copyThemeAssets(); err != nil {
		return err
	}

	return nil
}

// newBuilder constructs a Builder from the given options, deriving the site
// layout and loading the config and theme the rest of the build depends on
func newBuilder(opts BuildOptions) (*Builder, error) {
	paths := NewSitePaths(opts.SiteRoot, opts.DestDir)

	config, err := loadConfig(paths.Config)
	if err != nil {
		return nil, err
	}

	themeDir := filepath.Join(ThemesRoot, config.Theme)

	theme, err := loadTheme(themeDir, paths.Layouts)
	if err != nil {
		return nil, err
	}

	return &Builder{
		contentDir: paths.Content,
		themeDir:   themeDir,
		destDir:    paths.Dest,
		config:     config,
		theme:      theme,
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
