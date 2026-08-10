package site

import (
	"errors"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/notblankz/forge/internal/engine"
)

// @theme - the theme's template files. Content-hashed by concatenating each
// template's hash (deterministic WalkDir order), matching old forge's @theme
type themeData struct {
	theme      *template.Template
	shortcodes *Shortcodes
}

type themeNode struct {
	themeDir    string
	siteLayouts string
	contentDir  string
}

func (t themeNode) Hash(string) (string, error) {
	var sum strings.Builder
	for _, dir := range []string{filepath.Join(t.themeDir, "layouts"), t.siteLayouts} {

		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if d.IsDir() {
				return nil
			}
			h, err := engine.HashFile(path)
			if err != nil {
				return err
			}
			sum.WriteString(h)
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return engine.HashBytes([]byte(sum.String())), nil
}

func (t themeNode) Build(ctx *engine.BuildCtx, key string) (engine.Result, error) {
	cfg, err := ctx.Need(key, "@config")
	if err != nil {
		return engine.Result{}, err
	}
	md := cfg.Data.(configData).markdown

	theme, err := loadTheme(t.themeDir, t.siteLayouts)
	if err != nil {
		return engine.Result{}, err
	}

	shortcodes, err := loadShortcodes(t.themeDir, t.siteLayouts, t.contentDir, md)
	if err != nil {
		return engine.Result{}, err
	}

	return engine.Result{
		Data: themeData{
			theme:      theme,
			shortcodes: shortcodes,
		},
	}, nil
}

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

// ResolveThemeDir locates a theme from the site.toml `theme` value:
//   - an absolute path is used as-is
//   - a relative path is resolved against the site root
//   - a bare name maps to <siteRoot>/themes/<name>
func ResolveThemeDir(siteRoot, theme string) string {
	switch {
	case filepath.IsAbs(theme):
		return theme
	case strings.ContainsAny(theme, `/\`):
		return filepath.Join(siteRoot, theme)
	default:
		return filepath.Join(siteRoot, "themes", theme)
	}
}
