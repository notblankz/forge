package site

import (
	"html/template"
	"path/filepath"
	"strings"
)

// loadTheme parses all HTML templates in the given theme directory
// into a single template set / theme, keyed by the template filename
func loadTheme(dir string) (*template.Template, error) {

	theme, err := template.ParseGlob(filepath.Join(dir, "*.html"))
	if err != nil {
		return nil, err
	}

	return theme, nil
}

// selectTemplate returns the name of the template a page should render with by
// walking a cascade: 1) explicit frontmatter override -> 2) template matching
// the page's filename -> 3) generic page.html fallback
func selectTemplate(theme *template.Template, page Page) *template.Template {
	if page.Frontmatter.Template != "" {
		if t := theme.Lookup(page.Frontmatter.Template); t != nil {
			return t
		}
	}

	name := strings.TrimSuffix(filepath.Base(page.Path), filepath.Ext(page.Path)) + ".html"
	if t := theme.Lookup(name); t != nil {
		return t
	}

	// TODO: type-based selection (collection item -> post.html,
	// collection index -> listing.html) once collections exist

	return theme.Lookup("page.html")
}
