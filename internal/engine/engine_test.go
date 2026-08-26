package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/notblankz/forge/internal/timing"
)

// -- shared test helpers --

// fakeBuilder is a Builder assembled from two closures, so each test can shape
// hashing and building without declaring a new type.
type fakeBuilder struct {
	hashFn  func(key string) (string, error)
	buildFn func(ctx *BuildCtx, key string) (Result, error)
}

func (f fakeBuilder) Hash(key string) (string, error)                 { return f.hashFn(key) }
func (f fakeBuilder) Build(ctx *BuildCtx, key string) (Result, error) { return f.buildFn(ctx, key) }

// constHash is a Hash func that always returns h (a stable, non-erroring fingerprint).
func constHash(h string) func(string) (string, error) {
	return func(string) (string, error) { return h, nil }
}

// jsonEqual compares two raw JSON payloads by value, so re-indentation from a
// manifest round-trip doesn't count as a difference.
func jsonEqual(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatalf("unmarshal a: %v", err)
	}
	if err := json.Unmarshal(b, &y); err != nil {
		t.Fatalf("unmarshal b: %v", err)
	}
	return reflect.DeepEqual(x, y)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// AI Generated tests
var buildLog []string

func logBuild(key string) { buildLog = append(buildLog, key) }

// leaf node: its "source" is a value in a map; a missing key = deleted source.
type numBuilder struct{ vals map[string]string }

func (n numBuilder) Hash(key string) (string, error) {
	v, ok := n.vals[key]
	if !ok {
		return "", fs.ErrNotExist
	}
	return v, nil
}
func (n numBuilder) Build(_ *BuildCtx, key string) (Result, error) {
	logBuild(key)
	return Result{Outputs: []string{key}}, nil
}

// derived node: Needs its parts, has no source hash.
type sumBuilder struct{ parts map[string][]string }

func (sumBuilder) Hash(_ string) (string, error) { return "", nil }
func (s sumBuilder) Build(ctx *BuildCtx, key string) (Result, error) {
	logBuild(key)
	var outs []string
	for _, p := range s.parts[key] {
		r, err := ctx.Need(key, p)
		if err != nil {
			return Result{}, err
		}
		outs = append(outs, r.Outputs...)
	}
	return Result{Outputs: outs}, nil
}

func TestEngineIncremental(t *testing.T) {
	nums := numBuilder{vals: map[string]string{"@num:a": "1", "@num:b": "2", "@num:c": "3"}}
	parts := map[string][]string{
		"@sum:ab":  {"@num:a", "@num:b"},
		"@sum:all": {"@sum:ab", "@num:c"},
	}
	Register("@num", nums)
	Register("@sum", sumBuilder{parts: parts})

	targets := []string{"@sum:all"}

	run := func(name string, prev Manifest) Manifest {
		buildLog = nil
		m, err := Run(prev, targets, timing.NewTimer().Mark)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got := append([]string(nil), buildLog...)
		sort.Strings(got)
		fmt.Printf("%-9s built: %v\n", name, got)
		return m
	}
	wantBuilt := func(name string, want ...string) {
		got := append([]string(nil), buildLog...)
		sort.Strings(got)
		sort.Strings(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s: built %v, want %v", name, got, want)
		}
	}

	// 1. cold — everything builds
	m := run("cold", nil)
	wantBuilt("cold", "@num:a", "@num:b", "@num:c", "@sum:ab", "@sum:all")

	// 2. no change — nothing builds
	m = run("noop", m)
	wantBuilt("noop")

	// 3. change @num:b — b + the sums using it rebuild; a and c are reused
	nums.vals["@num:b"] = "9"
	m = run("change-b", m)
	wantBuilt("change-b", "@num:b", "@sum:ab", "@sum:all")

	// 4. delete @num:c (source gone + drop its reference) — @sum:all rebuilds without it
	delete(nums.vals, "@num:c")
	parts["@sum:all"] = []string{"@sum:ab"}
	m = run("delete-c", m)
	wantBuilt("delete-c", "@sum:all")
	if _, ok := m["@num:c"]; ok {
		t.Error("delete-c: @num:c should be gone from the manifest")
	}
}

// A shared dependency reached through two parents must be built exactly once —
// this is what the resultOf memo and singleflight buy us.
func TestEngineDiamondBuildsOnce(t *testing.T) {
	var leaf, midA, midB, top int

	Register("@leaf", fakeBuilder{
		constHash(""),
		func(_ *BuildCtx, key string) (Result, error) {
			leaf++
			return Result{Outputs: []string{key}}, nil
		},
	})
	Register("@mid", fakeBuilder{
		constHash(""),
		func(ctx *BuildCtx, key string) (Result, error) {
			if key == "@mid:a" {
				midA++
			} else {
				midB++
			}
			_, err := ctx.Need(key, "@leaf:x")
			return Result{Outputs: []string{key}}, err
		},
	})
	Register("@top", fakeBuilder{
		constHash(""),
		func(ctx *BuildCtx, key string) (Result, error) {
			top++
			if _, err := ctx.Need(key, "@mid:a"); err != nil {
				return Result{}, err
			}
			if _, err := ctx.Need(key, "@mid:b"); err != nil {
				return Result{}, err
			}
			return Result{Outputs: []string{key}}, nil
		},
	})

	if _, err := Run(nil, []string{"@top:t"}, nil); err != nil {
		t.Fatal(err)
	}
	if leaf != 1 {
		t.Errorf("shared leaf built %d times, want 1", leaf)
	}
	if midA != 1 || midB != 1 || top != 1 {
		t.Errorf("build counts a=%d b=%d top=%d, want 1 each", midA, midB, top)
	}
}

// A node whose output disappears from the manifest has its file removed from
// dist, while a live node's file is left alone.
func TestEngineOrphanCleanup(t *testing.T) {
	dir := t.TempDir()
	vals := map[string]string{"@art:a": "1", "@art:b": "1"}

	Register("@art", fakeBuilder{
		func(key string) (string, error) {
			v, ok := vals[key]
			if !ok {
				return "", fs.ErrNotExist
			}
			return v, nil
		},
		func(_ *BuildCtx, key string) (Result, error) {
			p := filepath.Join(dir, NodeID(key)+".html")
			if err := os.WriteFile(p, []byte(key), 0644); err != nil {
				return Result{}, err
			}
			return Result{Outputs: []string{p}}, nil
		},
	})

	fileA := filepath.Join(dir, "a.html")
	fileB := filepath.Join(dir, "b.html")

	m, err := Run(nil, []string{"@art:a", "@art:b"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !exists(fileA) || !exists(fileB) {
		t.Fatal("cold build should have written both files")
	}

	// delete b's source and stop targeting it — its output is now an orphan
	delete(vals, "@art:b")
	if _, err := Run(m, []string{"@art:a"}, nil); err != nil {
		t.Fatal(err)
	}
	if exists(fileB) {
		t.Error("orphaned output b.html should have been removed")
	}
	if !exists(fileA) {
		t.Error("live output a.html should have been kept")
	}
}

// Meta survives being persisted and is handed back verbatim when a clean node
// is served from the cache instead of rebuilt.
func TestEngineMetaRoundTrip(t *testing.T) {
	metaBytes := json.RawMessage(`{"title":"hi","n":3}`)
	srcVals := map[string]string{"@src:s": "v1"}
	docVals := map[string]string{"@doc2:d": "d1"}

	Register("@src", fakeBuilder{
		func(key string) (string, error) { return srcVals[key], nil },
		func(_ *BuildCtx, _ string) (Result, error) {
			// Outputs must be non-empty, or the engine won't cache-reuse this node.
			return Result{Outputs: []string{"src-out"}, Meta: metaBytes}, nil
		},
	})

	var seen json.RawMessage
	Register("@doc2", fakeBuilder{
		func(key string) (string, error) { return docVals[key], nil },
		func(ctx *BuildCtx, key string) (Result, error) {
			r, err := ctx.Need(key, "@src:s")
			if err != nil {
				return Result{}, err
			}
			seen = r.Meta
			return Result{Outputs: []string{"doc-out"}}, nil
		},
	})

	targets := []string{"@doc2:d"}

	m, err := Run(nil, targets, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, seen, metaBytes) {
		t.Fatalf("fresh build: doc saw Meta %s, want %s", seen, metaBytes)
	}

	// round-trip the manifest through disk
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := SaveManifest(path, m); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	// dirty only the doc; src stays clean and must be served from the loaded
	// manifest, carrying its persisted Meta through the cache-reuse path.
	seen = nil
	docVals["@doc2:d"] = "d2"
	if _, err := Run(loaded, targets, nil); err != nil {
		t.Fatal(err)
	}
	if seen == nil {
		t.Fatal("doc did not rebuild, so the reuse path was never exercised")
	}
	if !jsonEqual(t, seen, metaBytes) {
		t.Errorf("reused Meta = %s, want %s", seen, metaBytes)
	}
}

// Build errors, unregistered kinds, and non-deletion Hash errors all abort Run.
func TestEngineErrors(t *testing.T) {
	// a builder that fails propagates its error
	Register("@boom", fakeBuilder{
		constHash("h"),
		func(_ *BuildCtx, _ string) (Result, error) { return Result{}, errors.New("build failed") },
	})
	if _, err := Run(nil, []string{"@boom:x"}, nil); err == nil || !strings.Contains(err.Error(), "build failed") {
		t.Errorf("build error not propagated: %v", err)
	}

	// an unregistered kind is reported before anything is built
	if _, err := Run(nil, []string{"@ghost:z"}, nil); err == nil || !strings.Contains(err.Error(), "no builder") {
		t.Errorf("missing builder not reported: %v", err)
	}

	// a Hash error that isn't fs.ErrNotExist is a real failure, not a deletion
	Register("@badhash", fakeBuilder{
		func(string) (string, error) { return "", errors.New("hash boom") },
		func(_ *BuildCtx, _ string) (Result, error) { return Result{}, nil },
	})
	if _, err := Run(nil, []string{"@badhash:x"}, nil); err == nil || !strings.Contains(err.Error(), "hash boom") {
		t.Errorf("hash error not propagated: %v", err)
	}
}

// The diff helpers on their own: changedSet spots added/removed/rehashed nodes,
// and DirtySet walks that up the inputs graph to every dependent.
func TestDirtySetAndChangedSet(t *testing.T) {
	prev := Manifest{
		"@leaf:x": {Hash: "1"},
		"@mid:m":  {Hash: "m", Inputs: []string{"@leaf:x"}},
		"@top:t":  {Hash: "t", Inputs: []string{"@mid:m"}},
		"@iso:i":  {Hash: "i"},
		"@gone:g": {Hash: "g"},
	}
	curr := Manifest{
		"@leaf:x": {Hash: "2"}, // rehashed
		"@mid:m":  {Hash: "m", Inputs: []string{"@leaf:x"}},
		"@top:t":  {Hash: "t", Inputs: []string{"@mid:m"}},
		"@iso:i":  {Hash: "i"}, // unchanged, no path to the leaf
		"@new:n":  {Hash: "n"}, // added
		// @gone:g removed
	}

	changed := changedSet(prev, curr)
	wantChanged := map[string]struct{}{"@leaf:x": {}, "@new:n": {}, "@gone:g": {}}
	if !reflect.DeepEqual(changed, wantChanged) {
		t.Errorf("changedSet = %v, want %v", changed, wantChanged)
	}

	dirty := DirtySet(prev, curr)
	for _, k := range []string{"@leaf:x", "@mid:m", "@top:t"} {
		if _, ok := dirty[k]; !ok {
			t.Errorf("DirtySet missing %s — a leaf change should propagate up", k)
		}
	}
	if _, ok := dirty["@iso:i"]; ok {
		t.Error("DirtySet should not include a node with no path to the change")
	}
}

// SaveManifest -> LoadManifest is lossless, including raw Meta, and a missing
// file loads as (nil, nil).
func TestManifestRoundTrip(t *testing.T) {
	m := Manifest{
		"@page:x": {
			Kind:    "@page",
			Hash:    "h1",
			Inputs:  []string{"@config", "@theme"},
			Outputs: []string{"dist/x.html"},
			Meta:    json.RawMessage(`{"title":"x","date":"2026-01-01T00:00:00Z"}`),
		},
		"@config": {Kind: "@config", Hash: "h2"},
	}

	path := filepath.Join(t.TempDir(), "m.json")
	if err := SaveManifest(path, m); err != nil {
		t.Fatal(err)
	}
	got, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	// compare by re-marshaling: both compact identically even though the loaded
	// Meta was re-indented on disk.
	a, _ := json.Marshal(m)
	b, _ := json.Marshal(got)
	if !bytes.Equal(a, b) {
		t.Errorf("round-trip mismatch:\n saved  = %s\n loaded = %s", a, b)
	}

	if mm, err := LoadManifest(filepath.Join(t.TempDir(), "nope.json")); err != nil || mm != nil {
		t.Errorf("missing manifest should load as (nil, nil), got (%v, %v)", mm, err)
	}
}
