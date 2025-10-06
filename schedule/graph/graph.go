package graph

import (
	"context"
	"fmt"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
	"gonum.org/v1/gonum/graph/traverse"

	"tools/schedule/extender"
	"tools/schedule/node"
)

type Graph struct {
	engine    *simple.DirectedGraph
	extenders []extender.Extender
	nodes     map[string]node.Node
}

func New(extenders ...extender.Extender) *Graph {
	return &Graph{
		engine:    simple.NewDirectedGraph(),
		extenders: extenders,
		nodes:     make(map[string]node.Node),
	}
}

func (g *Graph) Register(_ context.Context, packages ...node.Package) error {
	g.engine = simple.NewDirectedGraph()

	for _, pkg := range packages {
		n := node.New(pkg)

		g.engine.AddNode(n)
		g.nodes[pkg.Name()] = n
	}

	for _, n := range g.nodes {
		for _, dep := range n.Package().Dependencies() {
			depNode, ok := g.nodes[dep.To()]
			if !ok {
				// TODO: error
				continue
			}

			g.engine.SetEdge(simple.Edge{F: depNode, T: n})
		}
	}

	// Detect cycles in dependency engine
	if len(topo.DirectedCyclesIn(g.engine)) > 0 {
		return fmt.Errorf("cycle detected")
	}

	return nil
}

type Diff struct {
	Enabled  []node.Package
	Disabled []node.Package
}

func (g *Graph) Calculate(_ context.Context) (Diff, error) {
	if g.engine == nil {
		return Diff{}, nil
	}

	var diff Diff
	err := g.traverse(func(n node.Node) error {
		// global always enabled
		if n.Package().Name() == "global" {
			n.SetEnabled(true, "Global")
			return nil
		}

		var changed bool

		defer func() {
			if changed {
				if n.IsEnabled() {
					diff.Enabled = append(diff.Enabled, n.Package())
					return
				}

				diff.Disabled = append(diff.Disabled, n.Package())
			}
		}()

		// Check parent dependencies first
		if reason := g.disabledByDependencies(n); len(reason) > 0 {
			changed = n.SetEnabled(false, reason)
			return nil
		}

		for _, ext := range g.extenders {
			if !n.IsEnabled() && ext.IsTerminator() {
				break
			}

			res, err := ext.Apply(n.Package())
			if err != nil {
				return err
			}

			if ext.IsTerminator() {
				if !res.Enabled {
					changed = n.SetEnabled(res.Enabled, res.Reason)
					break
				}

				continue
			}

			changed = n.SetEnabled(res.Enabled, res.Reason)
		}

		return nil
	})

	return diff, err
}

func (g *Graph) traverse(visit func(node node.Node) error) error {
	sorted, _ := topo.Sort(g.engine)
	for _, sortedNode := range sorted {
		if n, ok := sortedNode.(node.Node); ok {
			if err := visit(n); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Graph) disabledByDependencies(n node.Node) string {
	deps := make(map[string]node.Dependency, len(n.Package().Dependencies()))
	for _, dep := range n.Package().Dependencies() {
		deps[dep.To()] = dep
	}

	parents := graph.NodesOf(g.engine.To(n.ID()))
	for _, parent := range parents {
		parentNode := parent.(node.Node)
		parentName := parentNode.Package().Name()

		// If parent is disabled, disable this node
		if !parentNode.IsEnabled() {
			return "parent disabled: " + parentName
		}

		// Check version requirement if dependency exists
		dep, has := deps[parentName]
		if !has {
			continue
		}

		delete(deps, parentName)

		// Check version mismatch only if version is specified and not empty
		if dep.Version() != "" && dep.Version() != parentNode.Package().Version() {
			return "version mismatch: " + parentName + " (required: " + dep.Version() + ", found: " + parentNode.Package().Version() + ")"
		}
	}

	// Check for missing required dependencies
	for _, dep := range deps {
		// Skip optional dependencies
		if dep.IsOptional() {
			continue
		}

		// Check if required dependency exists in the engine
		if _, exists := g.nodes[dep.To()]; !exists {
			return "required dependency not found: " + dep.To()
		}
	}

	return ""
}

func (g *Graph) Dependents(_ context.Context, name string) []node.Package {
	n, ok := g.nodes[name]
	if !ok {
		return nil
	}

	g.engine.Node(n.ID())

	var result []node.Package
	t := &traverse.BreadthFirst{
		Visit: func(current graph.Node) {
			// Don't include the starting node
			if current.ID() != n.ID() {
				result = append(result, current.(node.Node).Package())
			}
		},
	}

	t.Walk(g.engine, n, nil)

	return result
}
