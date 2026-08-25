package site

import (
	"os"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/notblankz/forge/internal/engine"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"gopkg.in/yaml.v3"
)

type configData struct {
	config   SiteConfig
	markdown goldmark.Markdown
}

// @config - the site.yaml file. Content-hashed: any edit dirties the entire site
type configNode struct{ path string }

func (c configNode) Hash(string) (string, error) {
	return engine.HashFile(c.path)
}

func (c configNode) Build(*engine.BuildCtx, string) (engine.Result, error) {
	cfg, err := LoadConfig(c.path)
	if err != nil {
		return engine.Result{}, err
	}
	return engine.Result{
		Data: configData{
			config:   cfg,
			markdown: buildMarkdown(cfg),
		},
	}, nil
}

func buildMarkdown(cfg SiteConfig) goldmark.Markdown {
	ext := []goldmark.Extender{extension.GFM}
	if cfg.SyntaxHighlighting() {
		ext = append(ext, highlighting.NewHighlighting(
			highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
		))
	}
	return goldmark.New(goldmark.WithExtensions(ext...))
}

// SiteConfig is the entire site.yaml, kept generic so any key is reachable in
// templates as .Site.<key>. Typed accessors exist only for the fields forge
// itself consumes during the build
type SiteConfig map[string]any

func (c SiteConfig) Theme() string {
	s, _ := c["theme"].(string)
	return s
}

func (c SiteConfig) SyntaxHighlighting() bool {
	v, ok := c["syntax_highlighting"].(bool)
	return !ok || v
}

func (c SiteConfig) ImageSizes() []int {
	raw, ok := c["image_sizes"].([]any)
	if !ok {
		return nil
	}

	sizes := make([]int, 0, len(raw))
	for _, v := range raw {
		switch n := v.(type) {
		case int:
			sizes = append(sizes, n)
		case int64:
			sizes = append(sizes, int(n))
		case float64:
			sizes = append(sizes, int(n))
		}
	}

	return sizes
}

// LoadConfig reads and parses the site.yaml at path into SiteConfig
func LoadConfig(path string) (SiteConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := SiteConfig{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
