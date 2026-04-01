package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// SearchInput 搜索工具输入
type SearchInput struct {
	Query string `json:"query" jsonschema_description:"搜索查询内容"`
	Limit int    `json:"limit" jsonschema_description:"返回结果数量限制，默认为5"`
}

// SearchOutput 搜索工具输出
type SearchOutput struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Summary string `json:"summary"`
}

// NewSearchTool 创建搜索工具
func NewSearchTool() (tool.BaseTool, error) {
	return utils.InferTool(
		"web_search_mock",
		"mock搜索互联网获取信息",
		func(ctx context.Context, input *SearchInput) (*SearchOutput, error) {
			// 模拟搜索结果
			if input.Limit <= 0 {
				input.Limit = 5
			}

			// 这里是模拟实现，实际应用中应调用真实的搜索 API
			results := []SearchResult{
				{
					Title:   fmt.Sprintf("关于 %s 的详细信息", input.Query),
					URL:     "https://example.com/result1",
					Summary: fmt.Sprintf("这是关于 %s 的详细信息和解释...", input.Query),
				},
				{
					Title:   fmt.Sprintf("%s - 维基百科", input.Query),
					URL:     "https://wikipedia.org/example",
					Summary: fmt.Sprintf("%s 的定义、历史和相关背景...", input.Query),
				},
			}

			if len(results) > input.Limit {
				results = results[:input.Limit]
			}

			return &SearchOutput{
				Query:   input.Query,
				Results: results,
			}, nil
		})
}
