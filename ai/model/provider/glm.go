package provider

import (
	"aimc-go/ai/model"
	"context"
	"errors"
)

type Glm struct{}

func NewGlm() *Glm { return &Glm{} }

func (c *Glm) Generate(ctx context.Context, input model.AIInput) (*model.AIOutput, error) {
	switch input.GenType {
	case model.Text:
		return &model.AIOutput{Text: "[Gemini Text] " + input.Prompt}, nil
	case model.Image:
		return &model.AIOutput{URL: "https://cdn.example.com/gemini_image.png"}, nil
	default:
		return nil, errors.New("unsupported model type")
	}
}
