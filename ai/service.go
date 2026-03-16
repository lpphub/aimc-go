package ai

import (
	"aimc-go/ai/model"
	"aimc-go/ai/model/provider"
	"context"
	"errors"
)

type Service struct {
	providers map[string]model.AIClient
}

func NewAIService() *Service {
	return &Service{
		providers: map[string]model.AIClient{
			"openai": provider.NewOpenAI(),
			"gemini": provider.NewGemini(),
			"glm":    provider.NewGlm(),
		},
	}
}

func (s *Service) Generate(ctx context.Context, providerName string, input model.AIInput) (*model.AIOutput, error) {
	client, ok := s.providers[providerName]
	if !ok {
		return nil, errors.New("provider not found")
	}
	return client.Generate(ctx, input)
}
