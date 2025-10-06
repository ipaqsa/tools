package schedule

import (
	"context"

	"tools/schedule/extender"
	"tools/schedule/graph"
	"tools/schedule/node"
)

type Schedule struct {
	State   node.State
	Package node.Package
}

type Scheduler struct {
	graph *graph.Graph
}

func New(extenders ...extender.Extender) *Scheduler {
	return &Scheduler{
		graph: graph.New(extenders...),
	}
}

func (s *Scheduler) Register(ctx context.Context, packages ...node.Package) error {
	return s.graph.Register(ctx, packages...)
}

func (s *Scheduler) Schedule(ctx context.Context, name string) ([]Schedule, error) {
	diff, err := s.graph.Calculate(ctx)
	if err != nil {
		return nil, err
	}

	set := make(map[string]bool)
	schedules := make([]Schedule, 0)

	// packages to enable
	for _, enabled := range diff.Enabled {
		if _, has := set[enabled.Name()]; has {
			continue
		}

		set[enabled.Name()] = true
		schedules = append(schedules, Schedule{
			State:   node.StateEnable,
			Package: enabled,
		})
	}

	// packages to disable
	for _, disabled := range diff.Disabled {
		if _, has := set[disabled.Name()]; has {
			continue
		}

		set[disabled.Name()] = true
		schedules = append(schedules, Schedule{
			State:   node.StateDisable,
			Package: disabled,
		})
	}

	// packages to rerun
	for _, child := range s.graph.Dependents(ctx, name) {
		if _, has := set[child.Name()]; has {
			continue
		}

		set[child.Name()] = true
		schedules = append(schedules, Schedule{
			State:   node.StateUpdate,
			Package: child,
		})
	}

	return schedules, nil
}

// Each step waits previous
// Converge:
// global run - in global queue. Without scheduling
// calculate - in global queue. Without scheduling. This task calculate graph(skip global)
// disable package - in a package queue. Without scheduling
// ensureCRDs - in a package queue. Without scheduling
// enable critical - in a package queue. With scheduling(wait each other by order).
// enable functional - in a package queue. Without scheduling
// enable application - in a package queue. Without scheduling
