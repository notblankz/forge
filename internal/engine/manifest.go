package engine

import "github.com/notblankz/forge/internal/dag"

type ManifestEntry struct {
	Kind    string   `json:"kind"`
	Hash    string   `json:"hash,omitempty"`
	Inputs  []string `json:"inputs,omitempty"`
	Outputs []string `json:"outputs,omitempty"`
}

type Manifest map[string]ManifestEntry

// changedSet returns the nodes whose own fingerprint differs
// that is nodes added, removed or whose hash changed
func changedSet(prev, curr Manifest) map[string]struct{} {
	changed := map[string]struct{}{}

	// Go through the entire current manifest and find the changes in hash
	for key, entry := range curr {
		if prevEntry, ok := prev[key]; !ok || prevEntry.Hash != entry.Hash {
			changed[key] = struct{}{}
		}
	}

	// Check for nodes that are removed i.e. they exist in prev but not in curr
	for key := range prev {
		if _, ok := curr[key]; !ok {
			changed[key] = struct{}{}
		}
	}

	return changed
}

// graphFrom rebuilds the DAG from the union of both curr and prev Manifest
func graphFrom(prev, curr Manifest) *dag.Graph {
	g := dag.NewGraph()
	for _, manifest := range []Manifest{prev, curr} {
		for key, entry := range manifest {
			g.AddNode(key)
			for _, dep := range entry.Inputs {
				g.AddNode(dep)
				_ = g.AddEdge(key, dep)
			}
		}
	}
	return g
}

// DirtySet returns the nodes whose own hash changed, plus everything that
// indirectly or directly depends on them, walked through the inputs graph
func DirtySet(prev, curr Manifest) map[string]struct{} {
	return graphFrom(prev, curr).Dirty(changedSet(prev, curr))
}
