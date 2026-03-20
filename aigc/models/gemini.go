package models

import (
	"aimc-go/aigc"
	"context"
)

type Gemini struct{}

func (m *Gemini) ID() aigc.ModelID {
	return "gemini-3.0"
}

func (m *Gemini) Generate(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {

	return &aigc.GenerateResponse{
		Text: "generated text: " + req.Prompt,
	}, nil
}
