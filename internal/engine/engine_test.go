package engine

import (
	"fmt"
	"io/fs"
	"slices"
	"sort"
	"testing"
)

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
		m, err := Run(prev, targets)
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
