package prompts

var Templates = map[string]string{
	"marketing_copy":  MarketingCopyTemplate,
	"marketing_image": MarketingImageTemplate,
}

const MarketingCopyTemplate = `你是一位资深营销文案专家。请根据以下需求撰写营销文案：

需求：%s

要求：
- 语言简洁有力
- 突出卖点
- 适合社交媒体传播`

const MarketingImageTemplate = `Generate a professional marketing image for: %s

Style: modern, clean, professional, eye-catching
Aspect ratio: 16:9
Color scheme: vibrant and brand-appropriate`
