package site

import (
	"github.com/BurntSushi/toml"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/notblankz/forge/internal/engine"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
)

type configData struct {
	config   SiteConfig
	markdown goldmark.Markdown
}

// @config - the site.toml file. Content-hashed: any edit dirties the entire site
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
	if cfg.SyntaxHighlighting {
		ext = append(ext, highlighting.NewHighlighting(
			highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
		))
	}
	return goldmark.New(goldmark.WithExtensions(ext...))
}

type NavItem struct {
	Label string `toml:"label"`
	URL   string `toml:"url"`
}
type SiteConfig struct {
	Title              string    `toml:"title"`
	Theme              string    `toml:"theme"`
	NavbarLogo         string    `toml:"navbar_logo"`
	Nav                []NavItem `toml:"nav"`
	Social             []NavItem `toml:"social"`
	SyntaxHighlighting bool      `toml:"syntax_highlighting"`
}

// LoadConfig reads and parses the site.toml at path into a SiteConfig
func LoadConfig(path string) (SiteConfig, error) {
	config := SiteConfig{
		SyntaxHighlighting: true,
	}
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return SiteConfig{}, err
	}
	return config, nil
}
