package models

import (
	"aimc-go/aigc"
	"context"
)

type OpenAI struct{}

func (m *OpenAI) ID() aigc.ModelID {
	return "openai-gpt5.4"
}

func (m *OpenAI) Generate(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {

	return &aigc.GenerateResponse{
		Text: "generated text: " + req.Prompt,
	}, nil
}
