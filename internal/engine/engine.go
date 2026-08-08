package engine

import "strings"

// Result is whatever a Builder produces
type Result struct {
	Outputs []string // files writted under dist/
	Data    any      // optional payload for callers
}

// Builder knows how to fingerprint and (re)build one kind of node
type Builder interface {
	Hash(key string) (string, error)
	Build(ctx *BuildCtx, key string) (Result, error)
}

var registry = map[string]Builder{}

// Register wires a builder to a node kind
func Register(kind string, b Builder) {
	registry[kind] = b
}

// kindOf derives a node's kind from it's key
func kindOf(key string) string {
	if strings.HasPrefix(key, "@") {
		if i := strings.IndexByte(key, ':'); i >= 0 {
			return key[:i]
		}
		return key
	}

	return "page"
}

func builderFor(key string) (Builder, bool) {
	b, ok := registry[kindOf(key)]
	return b, ok
}
