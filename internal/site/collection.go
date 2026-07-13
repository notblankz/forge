package site

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

type Collection struct {
	Name  string
	Pages []Page
	Index *Page // index.md if present, else nil → auto-generate
}

func groupCollections(pages []Page, contentRoot string) (map[string]*Collection, error) {
	collections := make(map[string](*Collection))

	for _, page := range pages {
		rel, err := filepath.Rel(contentRoot, page.Path)
		if err != nil {
			return nil, err
		}

		segments := strings.Split(filepath.ToSlash(rel), "/")
		if len(segments) < 2 {
			continue
		}

		name := segments[0]
		if collections[name] == nil {
			collections[name] = &Collection{Name: name}
		}
		c := collections[name]

		if filepath.Base(page.Path) == "index.md" {
			p := page
			c.Index = &p
		} else {
			c.Pages = append(c.Pages, page)
		}
	}

	return collections, nil
}

func generateListingPage(c *Collection, config SiteConfig, theme *template.Template, destRoot string) error {
	type listingView struct {
		Site  SiteConfig
		Name  string
		Pages []Page
	}

	view := listingView{
		Site:  config,
		Name:  c.Name,
		Pages: c.Pages,
	}

	tmpl := theme.Lookup("listing.html")
	if tmpl == nil {
		return fmt.Errorf("listing: no listing.html template in theme")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, view); err != nil {
		return err
	}

	// write the generated listing html file data to destRoot/<name>/index.html
	// TODO: read output dir from buildOptions / site.toml instead of hardcoding "dist"
	outPath := filepath.Join(destRoot, c.Name, "index.html")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(outPath, buf.Bytes(), 0755)
}
