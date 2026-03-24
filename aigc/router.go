package aigc

type Router struct {
	routes map[TaskType]ModelID
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[TaskType]ModelID),
	}
}

func (r *Router) Register(task TaskType, model ModelID) {
	r.routes[task] = model
}

func (r *Router) Resolve(req *GenerateRequest) ModelID {
	if req.Model != "" {
		return req.Model
	}
	return r.routes[req.Task]
}
