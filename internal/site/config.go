package site

import (
	"github.com/BurntSushi/toml"
)

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

// loadConfig reads and parses the site.toml at path into a SiteConfig
func LoadConfig(path string) (SiteConfig, error) {
	config := SiteConfig{
		SyntaxHighlighting: true,
	}
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return SiteConfig{}, err
	}
	return config, nil
}
