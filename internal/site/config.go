package site

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type SiteConfig struct {
	Title string `toml:"title"`
	Theme string `toml:"theme"`
}

// loadConfig reads and parses site.toml from the content root into a SiteConfig
func loadConfig(contentRoot string) (SiteConfig, error) {
	path := filepath.Join(contentRoot, "site.toml")
	var config SiteConfig
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return SiteConfig{}, err
	}
	return config, nil
}
