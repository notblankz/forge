package dag

import "fmt"

type Graph struct {
	// We use a map with value as empty struct to simulate HashSets in Go
	Nodes      map[string]struct{}
	Edges      map[string][]string // Edges[A] = what A depends on
	Dependents map[string][]string // Dependents[B] = what depends on B
}

// NewGraph returns a Graph with empty node and edge sets.
func NewGraph() *Graph {
	return &Graph{
		Nodes:      make(map[string]struct{}),
		Edges:      make(map[string][]string),
		Dependents: make(map[string][]string),
	}
}

// AddNode adds nodeID to the node set
func (g *Graph) AddNode(nodeID string) {
	g.Nodes[nodeID] = struct{}{}
}

// AddEdge records a directed edge fromID -> toID; both nodes must already exist
func (g *Graph) AddEdge(dependent, dependency string) error {
	if _, ok := g.Nodes[dependent]; !ok {
		return fmt.Errorf("dag: unknown node %q", dependent)
	}
	if _, ok := g.Nodes[dependency]; !ok {
		return fmt.Errorf("dag: unknown node %q", dependency)
	}
	// Make an edge from Dependent -> Dependency
	g.Edges[dependent] = append(g.Edges[dependent], dependency)
	// Make an edge from Dependency -> Dependent
	g.Dependents[dependency] = append(g.Dependents[dependency], dependent)
	return nil
}

// Dirty returns the set of all the Node IDs that are dirty
// and need to be rebuilt
func (g *Graph) Dirty(seeds map[string]struct{}) map[string]struct{} {
	queue := make([]string, 0)
	visited := make(map[string]struct{})

	// add the seed to the queue and visited list
	for nodeID := range seeds {
		queue = append(queue, nodeID)
		visited[nodeID] = struct{}{}
	}

	// start the loop to build the dirty set
	for len(queue) != 0 {
		poppedNodeID := queue[0]
		queue = queue[1:]

		dirtyChildren := g.Dependents[poppedNodeID]
		for _, child := range dirtyChildren {
			if _, ok := visited[child]; !ok {
				visited[child] = struct{}{}
				queue = append(queue, child)
			}
		}
	}

	return visited
}
