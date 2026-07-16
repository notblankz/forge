package site

import "path/filepath"

// ThemesRoot is the directory forge loads themes from, relative to where the
// forge binary runs (not the user's site)
// TODO: Program-relative until themes are embedded via go:embed
const ThemesRoot = "themes"

// SitePaths resolves the on-disk directory layout for a site. It is the single
// source of truth for where content, overrides, output, and config live, so the
// builder and the dev server never derive these independently
type SitePaths struct {
	Content string // <root>/content   - markdown pages and assets/
	Layouts string // <root>/layouts   - site-level template overrides
	Dest    string // build output
	Config  string // <root>/site.toml
}

// NewSitePaths derives the site layout from siteRoot. destOverride, when
// non-empty, replaces the default output directory (<root>/dist)
// TODO: set destOverride being sent from the --output flag
func NewSitePaths(siteRoot, destOverride string) SitePaths {
	dest := destOverride
	if dest == "" {
		dest = filepath.Join(siteRoot, "dist")
	}
	return SitePaths{
		Content: filepath.Join(siteRoot, "content"),
		Layouts: filepath.Join(siteRoot, "layouts"),
		Dest:    dest,
		Config:  filepath.Join(siteRoot, "site.toml"),
	}
}
