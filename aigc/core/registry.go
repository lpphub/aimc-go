package core

import "fmt"

type Registry struct {
	models map[ModelID]Model
}

func NewRegistry() *Registry {
	return &Registry{models: make(map[ModelID]Model)}
}

func (r *Registry) Register(m Model) {
	r.models[m.ID()] = m
}

func (r *Registry) Get(id ModelID) (Model, error) {
	m, ok := r.models[id]
	if !ok {
		return nil, fmt.Errorf("model not found: %s", id)
	}
	return m, nil
}

func (r *Registry) List() []ModelID {
	ids := make([]ModelID, 0, len(r.models))
	for id := range r.models {
		ids = append(ids, id)
	}
	return ids
}
