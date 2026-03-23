package aigc

import (
	"context"
	"fmt"
)

type Client struct {
	registry *Registry
	router   *Router
}

func NewClient(reg *Registry, router *Router) *Client {
	return &Client{
		registry: reg,
		router:   router,
	}
}

func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	modelID := c.router.Resolve(req)
	if modelID == "" {
		return nil, fmt.Errorf("no model resolved for task: %s", req.Task)
	}

	model, err := c.registry.Get(modelID)
	if err != nil {
		return nil, err
	}

	return model.Generate(ctx, req)
}
