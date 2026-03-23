package x

import (
	"aimc-go/aigc"
	"aimc-go/aigc/models"
	"aimc-go/aigc/prompts"
	"context"
	"fmt"
)

var client *aigc.Client

type Config struct {
	OpenAIKey  string
	GeminiKey  string
	TextModel  aigc.ModelID // default: "openai-gpt4o"
	ImageModel aigc.ModelID // default: "gemini-2.0-flash"
}

func Init(ctx context.Context, cfg Config) error {
	reg := aigc.NewRegistry()

	if cfg.OpenAIKey != "" {
		reg.Register(models.NewOpenAI(cfg.OpenAIKey))
	}
	if cfg.GeminiKey != "" {
		gemini, err := models.NewGemini(ctx, cfg.GeminiKey)
		if err != nil {
			return fmt.Errorf("init gemini failed: %w", err)
		}
		reg.Register(gemini)
	}

	router := aigc.NewRouter()

	textModel := cfg.TextModel
	if textModel == "" {
		textModel = "openai-gpt4o"
	}
	router.SetDefault(aigc.TaskMarketingCopy, textModel)
	router.SetDefault(aigc.TaskGeneralText, textModel)

	imageModel := cfg.ImageModel
	if imageModel == "" {
		imageModel = "gemini-2.0-flash"
	}
	router.SetDefault(aigc.TaskMarketingImage, imageModel)

	client = aigc.NewClient(reg, router)
	return nil
}

func Generate(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("x not initialized, call x.Init() first")
	}
	return client.Generate(ctx, req)
}

func MarketingCopy(ctx context.Context, input string) (*aigc.GenerateResponse, error) {
	prompt, err := prompts.PromptMarketingCopy(input)
	if err != nil {
		return nil, err
	}
	return Generate(ctx, &aigc.GenerateRequest{
		Task:   aigc.TaskMarketingCopy,
		Prompt: prompt,
	})
}

func MarketingImage(ctx context.Context, input string) (*aigc.GenerateResponse, error) {
	prompt, err := prompts.PromptMarketingImage(input)
	if err != nil {
		return nil, err
	}
	return Generate(ctx, &aigc.GenerateRequest{
		Task:   aigc.TaskMarketingImage,
		Prompt: prompt,
	})
}
