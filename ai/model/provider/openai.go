package provider

import (
	"aimc-go/ai/model"
	"context"
	"errors"
)

type OpenAI struct{}

func NewOpenAI() *OpenAI { return &OpenAI{} }

func (c *OpenAI) Generate(ctx context.Context, input model.AIInput) (*model.AIOutput, error) {
	switch input.GenType {
	case model.Text:
		return &model.AIOutput{Text: "[OpenAI Text] " + input.Prompt}, nil
	case model.Image:
		return &model.AIOutput{URL: "https://cdn.example.com/openai_image.png"}, nil
	default:
		return nil, errors.New("unsupported model type")
	}
}
