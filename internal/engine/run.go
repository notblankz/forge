package engine

import (
	"errors"
	"fmt"
	"io/fs"
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

	// final new Manifest which will be filled after re-fingerprinting all the nodes
	refreshedManifest := Manifest{}

	// keys = last build's nodes plus this run's targets (deduped against prevManifest)
	keys := make([]string, 0, len(prevManifest)+len(targets))
	for key := range prevManifest {
		keys = append(keys, key)
	}

	for _, t := range targets {
		if _, ok := prevManifest[t]; !ok {
			keys = append(keys, t)
		}
	}

	// per node hashing outcome: the fresh hash or gone=true when the source has disappeared
	type hashResult struct {
		hash string
		gone bool
	}
	result := make([]hashResult, len(keys))

	// re-fingerprint every node concurrently - Each goroutine writes only result[i] with a
	// hashResult struct which can easily show if the key was hashed or deleted. Any other error
	// exits the entire group
	hg := new(errgroup.Group)
	hg.SetLimit(runtime.NumCPU())
	for i, key := range keys {
		hg.Go(func() error {
			b, ok := builderFor(key)
			if !ok {
				return fmt.Errorf("engine: no builder for %q", key)
			}
			h, err := b.Hash(key)
			if errors.Is(err, fs.ErrNotExist) {
				result[i] = hashResult{gone: true}
				return nil
			}
			if err != nil {
				return err
			}
			result[i] = hashResult{hash: h}
			return nil
		})
	}

	if err := hg.Wait(); err != nil {
		return nil, err
	}

	// merge the result of the above concurrent execution into the map serially
	// since Go maps are not concurrent safe for writes. Updates the hash for
	// all the nodes
	for i, key := range keys {
		if result[i].gone {
			delete(refreshedManifest, key)
			continue
		}
		entry := prevManifest[key]
		entry.Hash = result[i].hash
		entry.Kind = kindOf(key)
		refreshedManifest[key] = entry
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
