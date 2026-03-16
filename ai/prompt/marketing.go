package prompt

import "fmt"

func SellingPoints(product string) string {
	prompt := `
根据产品描述生成3个核心卖点。

产品描述：
%s

要求：
- brief
- 3个
- 20字以内
`
	return fmt.Sprintf(prompt, product)
}

func Copywriting(points string) string {
	prompt := `
根据卖点生成一段营销文案。

卖点：
%s

要求：
- 吸引人
- 适合社交媒体
- 100字以内
`
	return fmt.Sprintf(prompt, points)
}

func ImagePrompt(copy string) string {
	prompt := `
根据文案生成AI绘图prompt。

文案：
%s

要求：
- 适合广告海报
- 风格现代
- 20字以内
`
	return fmt.Sprintf(prompt, copy)
}
