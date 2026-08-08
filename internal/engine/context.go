package engine

import (
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"
)

// BuildCtx carries the per-build state every builder shares
// mu: mutex
// results: is a memo which allows each node to run only once
// inputs: is a map of node -> dependencies it uses
// sf: dedupes concurrent builds of the same key
type BuildCtx struct {
	mu      sync.Mutex
	results map[string]Result
	inputs  map[string]map[string]struct{}
	sf      singleflight.Group
}

func NewBuildCtx() *BuildCtx {
	return &BuildCtx{
		results: map[string]Result{},
		inputs:  map[string]map[string]struct{}{},
	}
}

// Need records that `from` depends on `dep`, builds `dep` exactly once
// and returns its result so that the caller can use it
func (ctx *BuildCtx) Need(from, dep string) (Result, error) {
	ctx.mu.Lock()
	set, ok := ctx.inputs[from]
	if !ok {
		set = map[string]struct{}{}
		ctx.inputs[from] = set
	}
	set[dep] = struct{}{}
	ctx.mu.Unlock()

	return ctx.build(dep)
}

// build produces a node's result, this is done only once per Site build
// using the memo (results) and singleflight for dedpuing concurrency (sf)
func (ctx *BuildCtx) build(key string) (Result, error) {
	ctx.mu.Lock()
	if r, ok := ctx.results[key]; ok {
		ctx.mu.Unlock()
		return r, nil
	}
	ctx.mu.Unlock()

	v, err, _ := ctx.sf.Do(key, func() (any, error) {
		b, ok := builderFor(key)
		if !ok {
			return Result{}, fmt.Errorf("engine: no builder registered for %q", key)
		}
		r, err := b.Build(ctx, key)
		if err != nil {
			return Result{}, err
		}
		ctx.mu.Lock()
		ctx.results[key] = r
		ctx.mu.Unlock()
		return r, nil
	})

	if err != nil {
		return Result{}, err
	}

	return v.(Result), err
}

// Inputs returns the recorded dependency edges of every node, which is
// then fed into the manifest for storing state
func (ctx *BuildCtx) Inputs() map[string][]string {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	out := map[string][]string{}

	for from, set := range ctx.inputs {
		for dep := range set {
			out[from] = append(out[from], dep)
		}
	}

	return out
}
