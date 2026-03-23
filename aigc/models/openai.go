package models

import (
	"aimc-go/aigc"
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAI struct {
	client *openai.Client
	model  string
}

func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{
		client: openai.NewClient(apiKey),
		model:  "gpt-4o",
	}
}

func (m *OpenAI) ID() aigc.ModelID {
	return "openai-gpt4o"
}

func (m *OpenAI) Generate(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {
	switch req.Task {
	case aigc.TaskMarketingImage:
		return m.generateImage(ctx, req)
	default:
		return m.generateText(ctx, req)
	}
}

func (m *OpenAI) generateText(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {
	resp, err := m.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: m.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: req.Prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai text generation failed: %w", err)
	}

	text := resp.Choices[0].Message.Content
	return &aigc.GenerateResponse{
		Text: text,
		Meta: map[string]any{
			"usage": resp.Usage,
		},
	}, nil
}

func (m *OpenAI) generateImage(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {
	size := openai.CreateImageSize1792x1024
	if s, ok := req.Params["size"].(string); ok {
		size = s
	}

	resp, err := m.client.CreateImage(ctx, openai.ImageRequest{
		Prompt: req.Prompt,
		Model:  openai.CreateImageModelDallE3,
		Size:   size,
		N:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("openai image generation failed: %w", err)
	}

	return &aigc.GenerateResponse{
		URL: resp.Data[0].URL,
	}, nil
}
