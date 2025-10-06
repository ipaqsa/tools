package node

import (
	"sync"
	"sync/atomic"
)

const (
	StateUpdate State = iota
	StateEnable
	StateDisable
)

type State int

var count atomic.Int64

type Node interface {
	ID() int64
	Package() Package

	SetEnabled(enabled bool, reason string) bool
	IsEnabled() bool
	Reason() string
}

type Package interface {
	Name() string
	Version() string
	Group() int

	Dependencies() []Dependency
}

type node struct {
	id  int64
	pkg Package

	mtx   sync.Mutex
	state state
}

type state struct {
	Enabled bool
	Reason  string
}

func New(pkg Package) Node {
	return &node{
		id:  count.Add(1),
		pkg: pkg,
		state: state{
			Enabled: false,
		},
	}
}

func (n *node) ID() int64 {
	return n.id
}

func (n *node) Package() Package {
	return n.pkg
}

func (n *node) IsEnabled() bool {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	return n.state.Enabled
}

func (n *node) Reason() string {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	return n.state.Reason
}

func (n *node) SetEnabled(enabled bool, reason string) bool {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	var diff bool
	if n.state.Enabled != enabled {
		diff = true
	}

	n.state.Enabled = enabled
	n.state.Reason = reason

	return diff
}
