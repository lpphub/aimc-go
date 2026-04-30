package tools

import (
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

func InitTools(cm model.BaseChatModel) ([]tool.BaseTool, error) {
	var tools []tool.BaseTool

	ragTool, err := BuildRAGTool(nil, cm)
	if err != nil {
		return nil, fmt.Errorf("build rag tool: %w", err)
	}
	tools = append(tools, ragTool)

	timeTool, err := NewTimeTool()
	if err != nil {
		return nil, fmt.Errorf("create time tool: %w", err)
	}
	tools = append(tools, timeTool)

	return tools, nil
}