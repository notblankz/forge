package site

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/notblankz/forge/internal/engine"
	"github.com/notblankz/forge/internal/timing"
)

type BuildOptions struct {
	SiteRoot string
	DestDir  string
	Timing   bool
}

func Build(opts BuildOptions) error {
	t := timing.NewTimer()
	start := time.Now()
	paths := NewSitePaths(opts.SiteRoot, opts.DestDir)

	cfg, err := LoadConfig(paths.Config)
	if err != nil {
		return err
	}
	themeDir := ResolveThemeDir(paths.Root, cfg.Theme)

	contentPaths, err := collectContent(paths.Content)
	if err != nil {
		return err
	}

	members, hasIndex := map[string][]string{}, map[string]bool{}
	for _, page := range contentPaths {
		rel, err := filepath.Rel(paths.Content, page)
		if err != nil {
			return err
		}
		seg := strings.SplitN(filepath.ToSlash(rel), "/", 2)
		if len(seg) < 2 {
			continue // standalone page, not in a collection
		}
		name := seg[0]
		if filepath.Base(page) == "index.md" {
			hasIndex[name] = true
		} else {
			members[name] = append(members[name], page)
		}
	}

	targets := []string{"@assets", "@theme-static"}
	for _, p := range contentPaths {
		targets = append(targets, "@page:"+p)
	}
	listingMembers := map[string][]string{}
	for name, mem := range members {
		if !hasIndex[name] { // auto-listing only when there's no index.md
			targets = append(targets, "@listing:"+name)
			listingMembers[name] = mem
		}
	}

	registerNodes(paths, themeDir, listingMembers)

	t.Mark("enumerate content")

	manifestPath := filepath.Join(paths.Root, ".forge-manifest.json")
	prev, err := engine.LoadManifest(manifestPath)
	if err != nil {
		return err
	}

	t.Mark("load manifest")

	curr, err := engine.Run(prev, targets, t.Mark)
	if err != nil {
		return err
	}
	if err := engine.SaveManifest(manifestPath, curr); err != nil {
		return err
	}

	t.Mark("save manifest")

	fmt.Printf("built in %s\n", time.Since(start))
	if opts.Timing {
		t.Report(os.Stdout)
	}

	return nil
}

// collectContent walks the content root recursively and returns the paths of
// all markdown (.md) files found in a slice
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
