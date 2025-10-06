package node

type Dependency struct {
	name     string
	version  string
	optional bool
}

type DependencyOption func(*Dependency)

func WithVersion(version string) DependencyOption {
	return func(dependency *Dependency) {
		dependency.version = version
	}
}

func WithOptional() DependencyOption {
	return func(dependency *Dependency) {
		dependency.optional = true
	}
}

func NewDependency(name string, opts ...DependencyOption) Dependency {
	dep := &Dependency{
		name: name,
	}

	for _, opt := range opts {
		opt(dep)
	}

	return *dep
}

func (d *Dependency) To() string {
	return d.name
}

func (d *Dependency) Version() string {
	return d.version
}

func (d *Dependency) IsOptional() bool {
	return d.optional
}
