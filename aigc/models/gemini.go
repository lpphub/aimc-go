package models

import (
	"aimc-go/aigc/core"
	"context"
	"encoding/base64"
	"fmt"

	"google.golang.org/genai"
)

type Gemini struct {
	client *genai.Client
	model  string
}

func NewGemini(ctx context.Context, apiKey string) (*Gemini, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &Gemini{
		client: client,
		model:  "gemini-2.0-flash",
	}, nil
}

func (m *Gemini) ID() core.ModelID {
	return core.ModelID(m.model)
}

func (m *Gemini) Generate(ctx context.Context, req *core.GenerateRequest) (*core.GenerateResponse, error) {
	switch req.Task {
	case core.TaskMarketingImage:
		return m.generateImage(ctx, req)
	default:
		return m.generateText(ctx, req)
	}
}

func (m *Gemini) generateText(ctx context.Context, req *core.GenerateRequest) (*core.GenerateResponse, error) {
	result, err := m.client.Models.GenerateContent(ctx, m.model, genai.Text(req.Prompt), nil)
	if err != nil {
		return nil, fmt.Errorf("gemini text generation failed: %w", err)
	}

	text := result.Text()
	return &core.GenerateResponse{
		Text: text,
		Meta: map[string]any{
			"usage": result.UsageMetadata,
		},
	}, nil
}

func (m *Gemini) generateImage(ctx context.Context, req *core.GenerateRequest) (*core.GenerateResponse, error) {
	result, err := m.client.Models.GenerateContent(ctx, "imagen-3.0-generate-002", genai.Text(req.Prompt), nil)
	if err != nil {
		return nil, fmt.Errorf("gemini image generation failed: %w", err)
	}

	if len(result.Candidates) > 0 && result.Candidates[0].Content != nil &&
		len(result.Candidates[0].Content.Parts) > 0 {
		part := result.Candidates[0].Content.Parts[0]
		if part.InlineData != nil && len(part.InlineData.Data) > 0 {
			encoded := base64.StdEncoding.EncodeToString(part.InlineData.Data)
			return &core.GenerateResponse{
				URL: fmt.Sprintf("data:%s;base64,%s", part.InlineData.MIMEType, encoded),
				Meta: map[string]any{
					"mime_type": part.InlineData.MIMEType,
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("no image data in response")
}
