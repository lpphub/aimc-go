package tools

import (
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

// InitTools 初始化所有内置工具。cm 是工具所需的模型（例如 RAG）。
func InitTools(cm model.BaseChatModel) ([]tool.BaseTool, error) {
	var tools []tool.BaseTool

	ragTool, err := BuildRAGTool(nil, cm)
	if err != nil {
		return nil, fmt.Errorf("build rag tool: %w", err)
	}
	tools = append(tools, ragTool)

	searchTool, err := NewSearchTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create search tool: %w", err)
	}
	tools = append(tools, searchTool)

	return tools, nil
}
