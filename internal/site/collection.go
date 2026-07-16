package site

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Collection struct {
	Name  string
	Pages []Page
	Index *Page // index.md if present, else nil → auto-generate
}

// groupCollections partitions pages into collections keyed by their top-level
// content directory. Pages that are located directly in the content root are standalone
// and skipped. A collection's index.md is stored as its Index rather than listed
// among its collection.Pages field
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

// generateListingPage renders a collection's auto-generated index using the
// theme's listing.html template and writes it to <destRoot>/<name>/index.html
func (b *Builder) generateListingPage(c *Collection) error {
	type listingView struct {
		CommonView
		Name  string
		Pages []Page
	}

	view := listingView{
		CommonView: CommonView{
			Site:      b.config,
			PageTitle: c.Name,
		},
		Name:  c.Name,
		Pages: c.Pages,
	}

	var buf bytes.Buffer
	err := b.theme.ExecuteTemplate(&buf, "listing", view)
	if err != nil {
		return fmt.Errorf("generate listing page for %q: %w", c.Name, err)
	}

	// write the generated listing html file data to destRoot/<name>/index.html
	outPath := filepath.Join(b.destRoot, c.Name, "index.html")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(outPath, buf.Bytes(), 0644)
}
