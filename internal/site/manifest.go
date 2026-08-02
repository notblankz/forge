package site

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/notblankz/forge/internal/dag"
)

type manifestEntry struct {
	Hash    string            `json:"hash,omitempty"`
	Outputs []string          `json:"outputs,omitempty"`
	Deps    map[string]string `json:"deps,omitempty"`
	Edges   []string          `json:"edges,omitempty"`
}

// hashBytes returns the hex-encoded SHA-256 of b
func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum)
}

// hashFile returns the hex-encoded SHA-256 of the file at path
func hashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return hashBytes(b), nil
}

// hashDir fingerprints a directory from its files' path, size, and mtime
// (metadata only). A missing directory yields an empty fingerprint.
func hashDir(dir string) (string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", err
	}

	var b strings.Builder

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		fmt.Fprintf(&b, "%s|%d|%d\n", path, info.Size(), info.ModTime().UnixNano())
		return nil
	})
	if err != nil {
		return "", err
	}

	return hashBytes([]byte(b.String())), nil
}

// buildManifestMap builds the current manifest: each page's content hash and
// output path (deps carried over from prev), plus @config and @theme hashes
func (b *Builder) buildManifestMap(pages []Page, collections map[string]*Collection, prev map[string]manifestEntry) (map[string]manifestEntry, error) {
	manifest := make(map[string]manifestEntry)

	// This takes care of all the pages
	for _, page := range pages {
		entry := manifestEntry{Hash: page.Hash, Outputs: []string{page.OutputPath}}
		if p, ok := prev[page.Path]; ok {
			entry.Deps = p.Deps // default: keep last build's folder notes
		}
		manifest[page.Path] = entry
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

	// This takes care of the auto-generated listings
	for name, c := range collections {
		if c.Index == nil {
			// Collect all the pages this collection lists
			members := make([]string, len(c.Pages))
			for i, p := range c.Pages {
				members[i] = p.Path
			}
			// Put them as edges in the manifest entry
			manifest["@listing:"+name] = manifestEntry{
				Outputs: []string{filepath.Join(b.destDir, name, "index.html")},
				Edges:   members,
			}
		}
	}

	return manifest, nil
}

func buildGraphFromManifest(prev, curr map[string]manifestEntry) (*dag.Graph, error) {
	// Create a new graph
	g := dag.NewGraph()

	// iterate over both curr and prev manifest to build a Union Graph
	for _, manifest := range []map[string]manifestEntry{prev, curr} {
		for id, entry := range manifest {
			// Add new node for every ID in manifest
			g.AddNode(id)

			// Add a Dependency Node to the graph
			// Add a edge from ID -> Dependency
			for _, dep := range entry.Edges {
				g.AddNode(dep)
				if err := g.AddEdge(id, dep); err != nil {
					return nil, err
				}
			}
		}
	}
	return g, nil
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

// pagesWithChangedDeps returns pages whose recorded asset-folder deps changed since
// prev, re-fingerprinting each folder; these seed dirty propagation
func (b *Builder) pagesWithChangedDeps(prev map[string]manifestEntry) (map[string]struct{}, error) {
	changed := make(map[string]struct{})
	cache := make(map[string]string)

	for id, entry := range prev {
		for dir, oldSnapshot := range entry.Deps {
			nowSnapshot, ok := cache[dir]
			if !ok {
				var err error
				nowSnapshot, err = hashDir(filepath.Join(b.contentDir, dir))
				if err != nil {
					return nil, err
				}
				cache[dir] = nowSnapshot
			}
			if nowSnapshot != oldSnapshot {
				changed[id] = struct{}{}
				break
			}
		}
	}
	return changed, nil
}

// recordRenderedDeps records fresh asset-folder deps for the pages just rebuilt; pages
// not rebuilt keep the deps carried forward by buildManifestMap
func (b *Builder) recordRenderedDeps(curr map[string]manifestEntry, rendered map[string][]string) error {
	// Fresh snapshots for the pages we just rebuilt
	for path, dirs := range rendered {
		entry := curr[path]
		entry.Deps = make(map[string]string, len(dirs))
		for _, dir := range dirs {
			snap, err := hashDir(filepath.Join(b.contentDir, dir))
			if err != nil {
				return err
			}
			entry.Deps[dir] = snap
		}
		curr[path] = entry
	}
	return nil
}
