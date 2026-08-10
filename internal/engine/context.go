package engine

import (
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"
)

// BuildCtx carries the per-build state shared by every builder.
//
//	prevManifest         last build's manifest - the source for reusing clean nodes
//	dirtySet             nodes that must rebuild this run
//	mu                   guards resultOf, builtManifestEntries, inputsOf
//	resultOf             memo of each resolved node's Result (built or reused) - build-once
//	builtManifestEntries fresh manifest entries for nodes that were actually rebuilt
//	inputsOf             node -> the deps it Needed (the graph, recorded by Need)
//	sf                   dedupes concurrent builds of the same key
type BuildCtx struct {
	prevManifest         Manifest
	dirtySet             map[string]struct{}
	mu                   sync.Mutex
	resultOf             map[string]Result
	builtManifestEntries map[string]ManifestEntry
	inputsOf             map[string]map[string]struct{}
	sf                   singleflight.Group
}

func NewBuildCtx(prevManifest Manifest, dirty map[string]struct{}) *BuildCtx {
	return &BuildCtx{
		prevManifest: prevManifest, dirtySet: dirty,
		resultOf:             map[string]Result{},
		builtManifestEntries: map[string]ManifestEntry{},
		inputsOf:             map[string]map[string]struct{}{},
	}
}

// Need records that `from` depends on `dep`, builds `dep` exactly once
// and returns its result so that the caller can use it
func (ctx *BuildCtx) Need(from, dep string) (Result, error) {
	ctx.mu.Lock()
	set, ok := ctx.inputsOf[from]
	if !ok {
		set = map[string]struct{}{}
		ctx.inputsOf[from] = set
	}
	set[dep] = struct{}{}
	ctx.mu.Unlock()

	return ctx.build(dep)
}

// build produces a node's result, this is done only once per Site build
// using the memo (results) and singleflight for deduping concurrency (sf)
func (ctx *BuildCtx) build(key string) (Result, error) {
	ctx.mu.Lock()
	if r, ok := ctx.resultOf[key]; ok {
		ctx.mu.Unlock()
		return r, nil
	}
	ctx.mu.Unlock()

	// if the node is clean AND known last build (from prevManifest)
	// reuse the outputs instead of running the builder. If the node has
	// no previous outputs (Like @config, @theme) make them fall through and build
	if _, isDirty := ctx.dirtySet[key]; !isDirty {
		if entry, ok := ctx.prevManifest[key]; ok && len(entry.Outputs) > 0 {
			r := Result{Outputs: entry.Outputs, Meta: entry.Meta}
			ctx.mu.Lock()
			ctx.resultOf[key] = r
			ctx.mu.Unlock()
			return r, nil
		}
	}

	// dirty or brand new nodes are built with a single flight call
	// and their ManifestEntry is assembled here and added to builtEntries
	v, err, _ := ctx.sf.Do(key, func() (any, error) {
		b, ok := builderFor(key)
		if !ok {
			return Result{}, fmt.Errorf("engine: no builder for %q", key)
		}

		r, err := b.Build(ctx, key)
		if err != nil {
			return Result{}, err
		}

		h, err := b.Hash(key)
		if err != nil {
			return Result{}, err
		}

		ctx.mu.Lock()
		ctx.resultOf[key] = r
		ctx.builtManifestEntries[key] = ManifestEntry{
			Kind:    kindOf(key),
			Hash:    h,
			Inputs:  flatten(ctx.inputsOf[key]),
			Outputs: r.Outputs,
			Meta:    r.Meta,
		}
		ctx.mu.Unlock()
		return r, nil
	})

	if err != nil {
		return Result{}, err
	}

	return v.(Result), nil
}

func flatten(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}
