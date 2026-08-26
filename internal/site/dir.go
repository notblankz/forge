package site

import (
	"path/filepath"

	"github.com/notblankz/forge/internal/engine"
)

// @dir:<sub> - an asset folder read via readDir. Metadata-hashed (path/size/mtime)
// so big images aren't re-read every build. Key "@dir:assets/gallery" fingerprints
// <contentDir>/assets/gallery.
type dirNode struct{ contentDir string }

func (d dirNode) Hash(key string) (string, error) {
	return engine.HashDir(filepath.Join(d.contentDir, engine.NodeID(key)))
}
func (dirNode) Build(*engine.BuildCtx, string) (engine.Result, error) {
	return engine.Result{}, nil
}
