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

func NewLLM(ctx context.Context) (model.ToolCallingChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  os.Getenv("API_KEY"),
		Model:   os.Getenv("MODEL"),
		BaseURL: os.Getenv("API_BASE_URL"),
	})
}
