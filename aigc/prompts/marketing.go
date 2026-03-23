package prompts

const MarketingCopyTemplate = `你是一位资深营销文案专家。请根据以下需求撰写营销文案：

需求：{{.Input}}

要求：
- 语言简洁有力
- 突出卖点
- 适合社交媒体传播`

const MarketingImageTemplate = `Generate a professional marketing image for: {{.Input}}

Style: modern, clean, professional, eye-catching
Aspect ratio: 16:9
Color scheme: vibrant and brand-appropriate`

func PromptMarketingCopy(input string) (string, error) {
	return NewPrompt(MarketingCopyTemplate, map[string]any{
		"Input": input,
	})
}

func PromptMarketingImage(input string) (string, error) {
	return NewPrompt(MarketingImageTemplate, map[string]any{
		"Input": input,
	})
}
