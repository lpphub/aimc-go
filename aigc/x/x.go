package x

import (
	"aimc-go/aigc"
	"aimc-go/aigc/models"
	"aimc-go/aigc/prompts"
	"context"
	"fmt"
)

var client *aigc.Client

func Init() {
	reg := aigc.NewRegistry()

	om := models.NewOpenAI("")
	gm, _ := models.NewGemini(context.Background(), "")
	reg.Register(om)
	reg.Register(gm)

	router := aigc.NewRouter()
	router.Register(aigc.TaskMarketingCopy, om.ID())
	router.Register(aigc.TaskGeneralText, om.ID())

	router.Register(aigc.TaskMarketingImage, gm.ID())

	client = aigc.NewClient(reg, router)
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
