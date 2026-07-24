package dag

import "fmt"

type Graph struct {
	// We use a map with value as empty struct to simulate HashSets in Go
	Nodes map[string]struct{}
	Edges map[string][]string
}

func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]struct{}),
		Edges: make(map[string][]string),
	}
}

func (g *Graph) AddNode(nodeID string) {
	g.Nodes[nodeID] = struct{}{}
}

func (g *Graph) AddEdge(fromID, toID string) error {
	if _, ok := g.Nodes[fromID]; !ok {
		return fmt.Errorf("dag: unknown node %q", fromID)
	}
	if _, ok := g.Nodes[toID]; !ok {
		return fmt.Errorf("dag: unknown node %q", toID)
	}
	g.Edges[fromID] = append(g.Edges[fromID], toID)
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

		dirtyChildren := g.Edges[poppedNodeID]
		for _, child := range dirtyChildren {
			if _, ok := visited[child]; !ok {
				visited[child] = struct{}{}
				queue = append(queue, child)
			}
		}
	}

	return visited
}
