package aigc

type Router struct {
	defaults map[TaskType]ModelID
}

func NewRouter() *Router {
	return &Router{
		defaults: make(map[TaskType]ModelID),
	}
}

func (r *Router) SetDefault(task TaskType, model ModelID) {
	r.defaults[task] = model
}

func (r *Router) Resolve(req *GenerateRequest) ModelID {
	if req.Model != "" {
		return req.Model
	}
	return r.defaults[req.Task]
}
