package site

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// loadTheme parses all HTML templates in the given theme directory
// into a single template set / theme, keyed by the template filename
// NOTE: theme is the entire collection of parsed template files in the themes/* directory
func loadTheme(dir string) (*template.Template, error) {

	fsys := os.DirFS(dir)
	theme, err := template.ParseFS(fsys, "*.html", "partials/*.html")
	if err != nil {
		return nil, err
	}

	return theme, nil
}

// selectTemplate returns the name of the template a page should render with,
// walking a cascade:
// 1) explicit frontmatter override
// 2) template matching the page's filename
// 3) generic "page" fallback
// NOTE: existence of the template is checked by ExecuteTemplate at call time
func selectTemplate(theme *template.Template, page Page) string {
	if page.Frontmatter.Template != "" {
		name := strings.TrimSuffix(page.Frontmatter.Template, ".html")
		if theme.Lookup(name) != nil {
			return name
		}
	}

	name := strings.TrimSuffix(filepath.Base(page.Path), filepath.Ext(page.Path))
	if theme.Lookup(name) != nil {
		return name
	}

	return "page"
}
