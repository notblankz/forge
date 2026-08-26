package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"runtime"

	"golang.org/x/sync/errgroup"
)

// Run performs one incremental build and returns the manifest to persist
// There are Four phases:
//
//	(1) re-fingerprint every known node into a "refreshed" snapshot,
//	(2) diff it against prevManifest to get the dirty set
//	(3) rebuild only the dirty targets (clean deps are served from cache)
//	(4) assemble the new manifest from the reused + freshly built nodes.
//		`targets` are the caller's output-producing nodes (pages, listings);
//		their dependencies are discovered on the fly via Need and never listed here
func Run(prevManifest Manifest, targets []string, mark func(string)) (Manifest, error) {

	markStep := func(label string) {
		if mark != nil {
			mark(label)
		}
	}

	if prevManifest == nil {
		prevManifest = Manifest{}
	}

	// copy the prevManifest into the refreshedManifest snapshot
	refreshedManifest := Manifest{}
	maps.Copy(refreshedManifest, prevManifest)

	// nodesToRecheck is the set of nodes we have to re-check for changes
	// nodesToRecheck includes all nodes in prevManifest and the targets
	// this function is currently building
	nodesToRecheck := map[string]struct{}{}
	for key := range prevManifest {
		nodesToRecheck[key] = struct{}{}
	}
	for _, key := range targets {
		nodesToRecheck[key] = struct{}{}
	}

	// go through each node that we have to recheck, calculate it's new hash
	// and update the hash in refreshedManifest to create the dirty set later
	for key := range nodesToRecheck {
		b, ok := builderFor(key)
		if !ok {
			return nil, fmt.Errorf("engine: no builder for %q", key)
		}

		h, err := b.Hash(key)
		if errors.Is(err, fs.ErrNotExist) {
			// Source file is gone. Dropping it from refreshedManifest while it
			// still exists in prevManifest is exactly how DirtySet reads a
			// deletion - which then propagates to whatever depended on i
			delete(refreshedManifest, key)
			continue
		}
		if err != nil {
			return nil, err
		}
		manifestEntry := prevManifest[key]
		manifestEntry.Hash = h
		manifestEntry.Kind = kindOf(key)
		refreshedManifest[key] = manifestEntry
	}

	markStep("hash refresh")

	dirty := DirtySet(prevManifest, refreshedManifest)

	markStep("dirty set")

	// Rebuild all the target nodes, skipping any non dirty nodes
	// ANy dependencies that the Node might need is pulled in through `Need`
	// where the Dirty ones are rebuilt and clean ones are served from cache
	ctx := NewBuildCtx(prevManifest, dirty)
	g := new(errgroup.Group)
	g.SetLimit(runtime.NumCPU())
	for _, target := range targets {
		if _, isDirty := dirty[target]; !isDirty {
			continue
		}
		g.Go(func() error {
			_, err := ctx.build(target)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Assemble the new manifest
	// BuildCtx.build stores the updated ManifestEntry into BuildCtx.builtManifestEntries
	// while building the node
	currManifest := Manifest{}
	for key, entry := range prevManifest {
		if _, isDirty := dirty[key]; !isDirty {
			currManifest[key] = entry
		}
	}
	for key, entry := range ctx.builtManifestEntries {
		currManifest[key] = entry
	}

	markStep("build")

	if err := cleanOrphans(prevManifest, currManifest); err != nil {
		return nil, err
	}

	markStep("clean orphans")

	return currManifest, nil
}

func cleanOrphans(prev, curr Manifest) error {
	kept := map[string]struct{}{}

	for _, entry := range curr {
		for _, output := range entry.Outputs {
			kept[output] = struct{}{}
		}
	}

	for _, entry := range prev {
		for _, output := range entry.Outputs {
			if _, ok := kept[output]; !ok {
				if err := os.Remove(output); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}
			}
		}
	}
	return nil
}
