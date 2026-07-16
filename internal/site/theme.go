package site

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// loadTheme parses all HTML templates at themeDir (themeDir/layouts/*.html and
// themeDir/layouts/partials/*.html) into a single template set / theme, keyed
// by the template filename, then merges in any overrides from siteLayoutsDir
// since a later {{define}} of the same name replaces an earlier one, an
// override just needs to be parsed after the theme for it to apply the user defined one
// instead of the one in the theme
// NOTE: theme is the entire collection of parsed template files in the themes/* directory
func loadTheme(themeDir, siteLayoutsDir string) (*template.Template, error) {
	fsys := os.DirFS(themeDir)
	theme, err := template.ParseFS(fsys, "layouts/*.html", "layouts/partials/*.html")
	if err != nil {
		return nil, err
	}

	return mergeOverrides(theme, siteLayoutsDir)
}

// mergeOverrides parses every *.html directly under dir and under dir/partials/
// into theme, if any exist. A no-op when dir has no matching files, since
// site-level overrides are optional
func mergeOverrides(theme *template.Template, dir string) (*template.Template, error) {
	var files []string
	for _, pattern := range []string{
		filepath.Join(dir, "*.html"),
		filepath.Join(dir, "partials", "*.html"),
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}

	if len(files) == 0 {
		return theme, nil
	}

	return theme.ParseFiles(files...)
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
