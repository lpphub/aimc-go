package aigc

import "context"

type Client struct {
	registry *Registry
}

func NewClient(reg *Registry) *Client {
	return &Client{
		registry: reg,
	}
}

func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	m, err := c.registry.Get(req.Model)
	if err != nil {
		return nil, err
	}

	return m.Generate(ctx, req)
}

func (c *Client) Text(ctx context.Context, model ModelID, prompt string) (*GenerateResponse, error) {
	return c.Generate(ctx, &GenerateRequest{
		Model:  model,
		Type:   ContentText,
		Prompt: prompt,
	})
}

func (c *Client) Image(ctx context.Context, model ModelID, prompt string) (*GenerateResponse, error) {
	return c.Generate(ctx, &GenerateRequest{
		Model:  model,
		Type:   ContentImage,
		Prompt: prompt,
	})
}
