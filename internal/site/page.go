package site

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Page struct {
	Path        string
	Body        string
	OutputPath  string
	Frontmatter Frontmatter
	URL         string // holds the web path of the page
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
func (b *Builder) loadPage(path string) (Page, error) {
	newPage := Page{}

	newPage.Path = path

	fm, body, err := extractFrontmatter(path)
	if err != nil {
		return Page{}, err
	}
	newPage.Body = body

	frontmatter, err := parseFrontmatter([]byte(fm))
	if err != nil {
		return Page{}, err
	}
	newPage.Frontmatter = frontmatter

	if err := newPage.resolvePaths(b.contentDir, b.destDir); err != nil {
		return Page{}, err
	}

	return newPage, nil
}

// extractFrontmatter reads a content file and separates its YAML
// frontmatter from the markdown body, returning them as raw strings.
// If the file has no frontmatter, the frontmatter return is empty and
// the whole file is returned as the body
func extractFrontmatter(path string) (string, string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}

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

	return "", "", fmt.Errorf("frontmatter: unclosed delimiter in %q", path)
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

// render converts the page's markdown body to HTML and returns it
func (b *Builder) renderPage(p Page) ([]byte, error) {
	type pageView struct {
		CommonView
		Page
		Content template.HTML
	}

	// Expand the page markdown and resolve the shortcodes + add TOKENs
	exp, err := b.shortcodes.Expand(p.Body)
	if err != nil {
		return nil, err
	}

	var fragmentBuf bytes.Buffer
	if err := b.markdown.Convert([]byte(exp.markdown), &fragmentBuf); err != nil {
		return nil, err
	}

	content := exp.Restore(fragmentBuf.String())

	view := pageView{
		CommonView: CommonView{
			Site:      b.config,
			PageTitle: p.Frontmatter.Title,
		},
		Page:    p,
		Content: template.HTML(content),
	}

	tmplName := selectTemplate(b.theme, p)

	var pageBuf bytes.Buffer
	if err := b.theme.ExecuteTemplate(&pageBuf, tmplName, view); err != nil {
		return nil, fmt.Errorf("render %q: %w", p.Path, err)
	}

	return pageBuf.Bytes(), nil
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
