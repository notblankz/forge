package site

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type BuildOptions struct {
	Root string
}

func Build(opts BuildOptions) error {

	paths, err := collectContent(opts.Root)
	if err != nil {
		return err
	}
	fmt.Println(paths)

	for _, path := range paths {
		extractFrontmatter(path)
	}

	return nil
}

func collectContent(root string) ([]string, error) {

	res := make([]string, 0)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {

		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(d.Name())
		if ext == ".md" {
			res = append(res, path)
		}

		return nil
	})

	return res, err
}

func extractFrontmatter(path string) ([]string, []string, error) {
	// This function splits the file content into frontmatter and body
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	cleanedContent := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(cleanedContent, "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, lines, nil
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			frontmatter := lines[1:i]
			body := lines[i+1:]
			return frontmatter, body, nil
		}
	}

	return nil, nil, fmt.Errorf("frontmatter: unclosed delimiter in %q", path)
}
