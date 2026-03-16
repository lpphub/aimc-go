package marketing

import (
	"aimc-go/ai"
	"aimc-go/ai/model"
	"aimc-go/ai/prompt"
	"context"
)

type Input struct {
	Model  string
	Prompt string
}

type Output struct {
	SellingPoints string
	Copywriting   string
	ImagePrompt   string
	ImageURL      string
}

type Workflow struct {
	aiService *ai.Service
}

func NewWorkflow(aiService *ai.Service) *Workflow {
	return &Workflow{aiService: aiService}
}

func (w *Workflow) Run(ctx context.Context, in Input) (*Output, error) {
	// 1. 卖点
	pointsPrompt := prompt.SellingPoints(in.Prompt)
	points, _ := w.aiService.Generate(ctx, in.Model, model.AIInput{
		GenType: model.Text,
		Prompt:  pointsPrompt,
	})

	// 2. 文案
	copyPrompt := prompt.Copywriting(points.Text)
	copywriting, _ := w.aiService.Generate(ctx, in.Model, model.AIInput{
		GenType: model.Text,
		Prompt:  copyPrompt,
	})

	// 3. 图片
	imagePrompt := prompt.ImagePrompt(copywriting.Text)
	imageURL, _ := w.aiService.Generate(ctx, in.Model, model.AIInput{
		GenType: model.Image,
		Prompt:  imagePrompt,
	})

	return &Output{
		SellingPoints: points.Text,
		Copywriting:   copywriting.Text,
		ImageURL:      imageURL.URL,
	}, nil
}
