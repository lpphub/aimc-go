package x

import (
	"aimc-go/aigc/core"
	"aimc-go/aigc/models"
	"aimc-go/aigc/prompts"
	"context"
	"fmt"
)

var client *core.Client

func Init() {
	reg := core.NewRegistry()

	om := models.NewOpenAI("")
	reg.Register(om)

	gm, err := models.NewGemini(context.Background(), "")
	if err == nil {
		reg.Register(gm)
	}

	router := core.NewRouter()
	router.Register(core.TaskMarketingCopy, om.ID())
	router.Register(core.TaskGeneralText, om.ID())

	if gm != nil {
		router.Register(core.TaskMarketingImage, gm.ID())
	}

	client = core.NewClient(reg, router)
}

func Generate(ctx context.Context, req *core.GenerateRequest) (*core.GenerateResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("x not initialized, call x.Init() first")
	}
	return client.Generate(ctx, req)
}

func MarketingCopy(ctx context.Context, input string) (*core.GenerateResponse, error) {
	prompt, err := prompts.PromptMarketingCopy(input)
	if err != nil {
		return nil, err
	}
	return Generate(ctx, &core.GenerateRequest{
		Task:   core.TaskMarketingCopy,
		Prompt: prompt,
	})
}

func MarketingImage(ctx context.Context, input string) (*core.GenerateResponse, error) {
	prompt, err := prompts.PromptMarketingImage(input)
	if err != nil {
		return nil, err
	}
	return Generate(ctx, &core.GenerateRequest{
		Task:   core.TaskMarketingImage,
		Prompt: prompt,
	})
}
