package extender

import (
	"tools/schedule/node"
)

type Extender interface {
	Apply(pkg node.Package) (Result, error)
	IsTerminator() bool
}

type Result struct {
	Enabled bool
	Reason  string
}
