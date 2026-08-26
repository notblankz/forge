package site

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/notblankz/forge/internal/engine"
	"gopkg.in/yaml.v3"
)

// page Node
type pageNode struct {
	contentDir string
	destDir    string
}

func (pageNode) Hash(key string) (string, error) {
	return engine.HashFile(engine.NodeID(key))
}

func (p pageNode) Build(ctx *engine.BuildCtx, key string) (engine.Result, error) {
	// Fetch the required inputs by Need-ing them from the context
	cfgRes, err := ctx.Need(key, "@config")
	if err != nil {
		return engine.Result{}, err
	}
	cfg := cfgRes.Data.(configData)

	themeRes, err := ctx.Need(key, "@theme")
	if err != nil {
		return engine.Result{}, err
	}
	th := themeRes.Data.(themeData)

	// load and parse the page file
	page, err := loadPage(engine.NodeID(key), p.contentDir, p.destDir)
	if err != nil {
		return engine.Result{}, err
	}

	// expand shortcodes + markdown fragment + record asset folder this page readDir'ed
	exp, dirs, err := th.shortcodes.Expand(ctx, key, page.Body)
	if err != nil {
		return engine.Result{}, err
	}

	// Run through each readDir'ed dir of this page and call Need on it to make an edge in Graph
	for _, dir := range dirs {
		if _, err := ctx.Need(key, "@dir:"+dir); err != nil {
			return engine.Result{}, err
		}
	}

	// Convert markdown to HTML using goldmark
	var frag bytes.Buffer
	if err := cfg.markdown.Convert([]byte(exp.markdown), &frag); err != nil {
		return engine.Result{}, err
	}
	content := exp.Restore(frag.String())

	// render through the theme template
	type pageView struct {
		Site        SiteConfig
		Frontmatter Frontmatter
		URL         string
		Content     template.HTML
	}

	view := pageView{
		Site:        cfg.config,
		Frontmatter: page.Frontmatter,
		URL:         page.URL,
		Content:     template.HTML(content),
	}

	var out bytes.Buffer
	if err := th.theme.ExecuteTemplate(&out, selectTemplate(th.theme, page), view); err != nil {
		return engine.Result{}, err
	}

	// write the file
	if err := page.write(out.Bytes()); err != nil {
		return engine.Result{}, err
	}

	// serialise the Page metadata
	meta, err := json.Marshal(page.Metadata())
	if err != nil {
		return engine.Result{}, err
	}

	return engine.Result{
		Outputs: []string{page.OutputPath},
		Meta:    meta,
	}, nil
}

// Page Node Helpers

// -- Frontmatter --

// Frontmatter is a page's full front matter, generic so any field is reachable
// in templates as .Frontmatter.<key>. Typed accessors cover only what forge reads
type Frontmatter map[string]any

func (f Frontmatter) Title() string {
	s, _ := f["title"].(string)
	return s
}

func (f Frontmatter) Template() string {
	s, _ := f["template"].(string)
	return s
}

func (f Frontmatter) Date() time.Time {
	switch v := f["date"].(type) {
	case time.Time:
		return v
	case string:
		for _, layout := range []string{time.RFC3339, "2006-01-02"} {
			if t, err := time.Parse(layout, v); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// -- listing metadata --

// PageMetadata is the flat, listing-facing metadata: title/date/url plus opted-in
// fields. Stored in Meta and handed to listings — never used to render a page
type PageMetadata map[string]any

// UnmarshalJSON decodes persisted metadata and restores the date, which JSON
// stores as a string - so listings can call .Format/.IsZero on it directly
func (m *PageMetadata) UnmarshalJSON(data []byte) error {
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if s, ok := raw["date"].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			raw["date"] = t
		}
	}
	*m = raw
	return nil
}

// -- internal page representation --
type Page struct {
	Path        string
	Body        string
	OutputPath  string
	URL         string // holds the web path of the page
	Frontmatter Frontmatter
}

func (p Page) Metadata() PageMetadata {
	m := PageMetadata{
		"title": p.Frontmatter.Title(),
		"date":  p.Frontmatter.Date(),
		"url":   p.URL,
	}

	names, _ := p.Frontmatter["metadata"].([]any)
	for _, n := range names {
		key, ok := n.(string)
		if !ok || key == "title" || key == "date" || key == "url" {
			continue
		}
		if v, ok := p.Frontmatter[key]; ok {
			m[key] = v
		}
	}

	return m
}

// loadPage reads a content file and assembles it into a Page,
// extracting and parsing its frontmatter and body.
func loadPage(path, contentDir, destDir string) (Page, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Page{}, err
	}

	fm, body, err := extractFrontmatter(content)
	if err != nil {
		return Page{}, fmt.Errorf("%q: %w", path, err)
	}

	frontmatter, err := parseFrontmatter([]byte(fm))
	if err != nil {
		return Page{}, err
	}

	if frontmatter.Title() == "" {
		base := filepath.Base(path)
		frontmatter["title"] = strings.NewReplacer("-", " ", "_", " ").Replace(strings.TrimSuffix(base, filepath.Ext(base)))
	}

	page := Page{
		Path:        path,
		Body:        body,
		Frontmatter: frontmatter,
	}

	if err := page.resolvePaths(contentDir, destDir); err != nil {
		return Page{}, err
	}

	return page, nil
}

// extractFrontmatter accepts a byte slice of content, separates its YAML
// frontmatter from the markdown body, returning them as raw strings.
// If the file has no frontmatter, the frontmatter return is empty and
// the whole file is returned as the body
func extractFrontmatter(content []byte) (string, string, error) {

	cleanedContent := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(cleanedContent, "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", strings.Join(lines, "\n"), nil
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			frontmatter := lines[1:i]
			body := lines[i+1:]
			return strings.Join(frontmatter, "\n"), strings.Join(body, "\n"), nil
		}
	}

	return "", "", errors.New("frontmatter: unclosed delimiter")
}

// parseFrontmatter unmarshals raw YAML frontmatter into a Frontmatter struct
func parseFrontmatter(raw []byte) (Frontmatter, error) {
	res := Frontmatter{}
	err := yaml.Unmarshal(raw, &res)
	if err != nil {
		return Frontmatter{}, err
	}
	return res, nil
}

// write saves the given HTML content to the page's resolved output path,
// creating parent directories as needed
func (p *Page) write(content []byte) error {
	// 1) create file from p.OutputPath
	// 2) Write the content into the file
	if err := os.MkdirAll(filepath.Dir(p.OutputPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(p.OutputPath, content, 0644)
}

// resolvePaths sets the page's output path and URL
//
//	home.md       : dist/index.html            (/)
//	resume.md     : dist/resume.html           (/resume)
//	blog/post.md  : dist/blog/post.html        (/blog/post)
//	blog/index.md : dist/blog/index.html       (/blog/)
func (p *Page) resolvePaths(contentDir, destDir string) error {
	rel, err := filepath.Rel(contentDir, p.Path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(rel)
	base := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	slashDir := filepath.ToSlash(dir)

	var outRel string
	switch base {
	case "home":
		outRel = "index.html"
		p.URL = "/"
	case "index":
		outRel = filepath.Join(dir, "index.html")
		if slashDir == "." {
			p.URL = "/"
		} else {
			p.URL = "/" + slashDir + "/"
		}
	default:
		outRel = filepath.Join(dir, base+".html")
		if slashDir == "." {
			p.URL = "/" + base
		} else {
			p.URL = "/" + slashDir + "/" + base
		}
	}

	p.OutputPath = filepath.Join(destDir, outRel)
	return nil
}
