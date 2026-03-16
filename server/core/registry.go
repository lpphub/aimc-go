package core

type Registry struct {
	modules []Module
}

func NewRegistry() *Registry {
	return &Registry{
		modules: make([]Module, 0),
	}
}

func (r *Registry) Register(modules ...Module) {
	r.modules = append(r.modules, modules...)
}

func (r *Registry) Modules() []Module {
	return r.modules
}