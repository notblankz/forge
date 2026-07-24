package site

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type manifestEntry struct {
	Hash   string `json:"hash"`
	Output string `json:"output"`
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum)
}

func hashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return hashBytes(b), nil
}

func (b *Builder) buildManifestMap(pages []Page) (map[string]manifestEntry, error) {
	manifest := make(map[string]manifestEntry)

	// This takes care of all the pages
	for _, page := range pages {
		manifest[page.Path] = manifestEntry{
			Hash:   page.Hash,
			Output: page.OutputPath,
		}
	}

	// This takes care of the @theme folder files
	var themeSum strings.Builder
	staticDir := filepath.Join(b.themeDir, "static")
	err := filepath.WalkDir(b.themeDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path == staticDir {
				return filepath.SkipDir
			}
			return nil
		}

		sum, err := hashFile(path)
		if err != nil {
			return err
		}
		themeSum.WriteString(sum)

		return nil
	})
	if err != nil {
		return nil, err
	}
	manifest["@theme"] = manifestEntry{Hash: hashBytes([]byte(themeSum.String()))}

	// This takes care of the @config site.toml file
	configSum, err := hashFile(filepath.Join(b.siteRoot, "site.toml"))
	if err != nil {
		return nil, err
	}
	manifest["@config"] = manifestEntry{Hash: configSum}

	return manifest, nil
}

// loadManifest unmarshals the saved .forge-manifest.json (if any)
// and returns that as a map[string]manifestEntry
func (b *Builder) loadManifest() (map[string]manifestEntry, error) {
	var savedManifest map[string]manifestEntry
	manifestPath := filepath.Join(b.siteRoot, ".forge-manifest.json")

	content, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	err = json.Unmarshal(content, &savedManifest)
	if err != nil {
		return nil, err
	}

	return savedManifest, nil
}

// saveManifest takes the built manifest m and stores it
// on a local file at <root>/.forge-manifest.json
func (b *Builder) saveManifest(m map[string]manifestEntry) error {
	content, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(b.siteRoot, ".forge-manifest.json")
	if err := os.WriteFile(manifestPath, content, 0644); err != nil {
		return err
	}

	return nil
}

// diffManifests compares the previous build's manifest against the current one
// and returns the set of node IDs whose inputs changed - those added, removed,
// or whose hash differs. The returned set is used to seed the dirty propogation using DAG
func diffManifests(prev, curr map[string]manifestEntry) map[string]struct{} {
	changed := make(map[string]struct{})

	// Added or modified: present in curr, but missing from prev or with a
	// different hash
	for id, entry := range curr {
		if prevEntry, ok := prev[id]; !ok || prevEntry.Hash != entry.Hash {
			changed[id] = struct{}{}
		}
	}

	// Removed: present in prev but gone from curr
	for id := range prev {
		if _, ok := curr[id]; !ok {
			changed[id] = struct{}{}
		}
	}

	return changed
}

// deleteRemovedOutputs removes the rendered HTML of any page that was in the
// previous manifest but is gone from the current one (i.e. its source file was
// deleted), so stale pages don't linger in the output directory. This works on
// input-only nodes (@config/@theme) have no Output and are skipped.
func (b *Builder) deleteRemovedOutputs(prev, curr map[string]manifestEntry) error {
	for id, entry := range prev {
		if _, ok := curr[id]; ok {
			continue // still present
		}
		if entry.Output == "" {
			continue // not a page (@content/@listing), nothing on disk to remove
		}
		if err := os.Remove(entry.Output); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
