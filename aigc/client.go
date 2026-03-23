package aigc

import (
	"aimc-go/aigc/prompts"
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

	req.Prompt = c.buildPrompt(req)
	return model.Generate(ctx, req)
}

func (c *Client) MarketingCopy(ctx context.Context, input string) (*GenerateResponse, error) {
	return c.Generate(ctx, &GenerateRequest{
		Task:   TaskMarketingCopy,
		Prompt: input,
	})
}

func (c *Client) MarketingImage(ctx context.Context, input string) (*GenerateResponse, error) {
	return c.Generate(ctx, &GenerateRequest{
		Task:   TaskMarketingImage,
		Prompt: input,
	})
}

func (c *Client) buildPrompt(req *GenerateRequest) string {
	if tmpl, ok := prompts.Templates[string(req.Task)]; ok {
		return fmt.Sprintf(tmpl, req.Prompt)
	}
	return req.Prompt
}
