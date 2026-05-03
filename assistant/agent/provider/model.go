package provider

import (
	"context"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load(".env")
}

type Provider interface {
	NewChatModel(ctx context.Context) (model.ToolCallingChatModel, error)
}

type OpenAIProvider struct {
	APIKey  string
	Model   string
	BaseURL string
}

func (p *OpenAIProvider) NewChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  p.APIKey,
		Model:   p.Model,
		BaseURL: p.BaseURL,
		ByAzure: false,
	})
}

func NewProviderFromEnv() *OpenAIProvider {
	return &OpenAIProvider{
		APIKey:  os.Getenv("API_KEY"),
		Model:   os.Getenv("MODEL"),
		BaseURL: os.Getenv("API_BASE_URL"),
	}
}