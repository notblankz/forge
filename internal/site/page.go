package site

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Page struct {
	Path        string
	Body        string
	Frontmatter Frontmatter
}

type Frontmatter struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

func loadPage(path string) (Page, error) {
	// loadPage reads a content file and assembles it into a Page,
	// extracting and parsing its frontmatter and body.
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
	return newPage, nil
}

func extractFrontmatter(path string) (string, string, error) {
	// extractFrontmatter reads a content file and separates its YAML
	// frontmatter from the markdown body, returning them as raw strings.
	// If the file has no frontmatter, the frontmatter return is empty and
	// the whole file is returned as the body
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

func parseFrontmatter(raw []byte) (Frontmatter, error) {
	// This function takes the raw Frontmatter byte slice
	// and unmarshalls it into a Frontmatter Struct
	res := Frontmatter{}
	err := yaml.Unmarshal(raw, &res)
	if err != nil {
		return Frontmatter{}, err
	}
	return res, nil
}
