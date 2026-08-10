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
	exp, dirs, err := th.shortcodes.Expand(page.Body)
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
		CommonView
		Page
		Content template.HTML
	}

	view := pageView{
		CommonView: CommonView{
			Site:      cfg.config,
			PageTitle: page.Frontmatter.Title,
		},
		Page:    page,
		Content: template.HTML(content),
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
	meta, err := json.Marshal(PageMeta{
		Title:       page.Frontmatter.Title,
		Date:        page.Frontmatter.Date,
		Description: page.Frontmatter.Description,
		URL:         page.URL,
	})
	if err != nil {
		return engine.Result{}, err
	}

	return engine.Result{
		Outputs: []string{page.OutputPath},
		Meta:    meta,
	}, nil
}

// Page Node Helpers
type Page struct {
	Path        string
	Body        string
	OutputPath  string
	Frontmatter Frontmatter
	URL         string // holds the web path of the page
	Hash        string
}

type PageMeta struct {
	Title       string
	Date        time.Time
	Description string
	URL         string
}

type Frontmatter struct {
	Date        time.Time `taml:"date"`
	Title       string    `yaml:"title"`
	Description string    `yaml:"description"`
	Template    string    `yaml:"template"`
}

type CommonView struct {
	Site      SiteConfig
	PageTitle string // text for the <title> tag, decided per view
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

// resolvePaths sets the page's output path and URL, mapping the source path
// (relative to contentDir) into destDir using clean-URL layout:
//
//	home.md       : dist/index.html            (/)
//	resume.md     : dist/resume/index.html     (/resume/)
//	blog/post.md  : dist/blog/post/index.html  (/blog/post/)
//	blog/index.md : dist/blog/index.html       (/blog/)
func (p *Page) resolvePaths(contentDir, destDir string) error {
	rel, err := filepath.Rel(contentDir, p.Path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(rel)
	base := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))

	var outRel string
	switch base {
	case "home":
		outRel = "index.html"
	case "index":
		outRel = filepath.Join(dir, "index.html")
	default:
		outRel = filepath.Join(dir, base, "index.html")
	}

	p.OutputPath = filepath.Join(destDir, outRel)

	// Remove index.html and keep only dir/base/ as the URL
	urlPath := filepath.ToSlash(filepath.Dir(outRel))
	if urlPath == "." {
		p.URL = "/"
	} else {
		p.URL = "/" + urlPath + "/"
	}

	return nil
}
