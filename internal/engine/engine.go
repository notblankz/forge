package engine

import (
	"encoding/json"
	"strings"
)

// Result is whatever a Builder produces
type Result struct {
	Outputs []string        // files writted under dist/
	Data    any             // optional payload for callers
	Meta    json.RawMessage // small serialable facts, persister and reused across builds
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

// kindOf: "@page:content/x.md" -> "@page";  "@config" -> "@config"
func kindOf(key string) string {
	if before, _, ok := strings.Cut(key, ":"); ok {
		return before
	}
	return key
}

// NodeID returns the part after the kind: "@page:content/x.md" -> "content/x.md";
func NodeID(key string) string {
	if _, after, ok := strings.Cut(key, ":"); ok {
		return after
	}
	return ""
}

func builderFor(key string) (Builder, bool) {
	b, ok := registry[kindOf(key)]
	return b, ok
}
