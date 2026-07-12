package site

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

type Page struct {
	Path        string
	Body        string
	OutputPath  string
	Frontmatter Frontmatter
}

type Frontmatter struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Template    string `yaml:"template"`
}

// loadPage reads a content file and assembles it into a Page,
// extracting and parsing its frontmatter and body.
func loadPage(path string, contentRoot string) (Page, error) {
	newPage := Page{}

	newPage.Path = path

	f, b, err := extractFrontmatter(path)
	if err != nil {
		return Page{}, err
	}
	newPage.Body = b

	frontmatter, err := parseFrontmatter([]byte(f))
	if err != nil {
		return Page{}, err
	}
	newPage.Frontmatter = frontmatter

	// TODO: read output dir from buildOptions / site.toml instead of hardcoding "dist"
	if err := newPage.resolveOutputPath(contentRoot, "dist"); err != nil {
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

// parseFrontmatter unmarshals eaw YAML frontmatter into a Frontmatter struct
func parseFrontmatter(raw []byte) (Frontmatter, error) {
	res := Frontmatter{}
	err := yaml.Unmarshal(raw, &res)
	if err != nil {
		return Frontmatter{}, err
	}
	return res, nil
}

// render converts the page's markdown body to HTML and returns it
func (p *Page) render(theme *template.Template) ([]byte, error) {
	type pageView struct {
		Page
		Content template.HTML
	}

	var fragmentBuf bytes.Buffer
	if err := goldmark.Convert([]byte(p.Body), &fragmentBuf); err != nil {
		return nil, err
	}

	view := pageView{
		Page:    *p,
		Content: template.HTML(fragmentBuf.String()),
	}

	tmpl := selectTemplate(theme, *p)
	if tmpl == nil {
		return nil, fmt.Errorf("render: no template found for %q (missing page.html?)", p.Path)
	}

	var pageBuf bytes.Buffer
	if err := tmpl.Execute(&pageBuf, view); err != nil {
		return nil, err
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

// resolveOutputPath sets p.OutputPath by mapping the source path from
// the content root into destRoot, swapping .md for .html
func (p *Page) resolveOutputPath(contentRoot, destRoot string) error {
	rel, err := filepath.Rel(contentRoot, p.Path)
	if err != nil {
		return err
	}
	rel = strings.TrimSuffix(rel, filepath.Ext(rel)) + ".html"
	p.OutputPath = filepath.Join(destRoot, rel)
	return nil
}
