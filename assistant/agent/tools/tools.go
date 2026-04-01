package tools

import (
	"aimc-go/assistant/agent/llm"
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
)

type Registry struct {
	tools []tool.BaseTool
}

// NewRegistry 创建工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make([]tool.BaseTool, 0),
	}
}

// Register 注册工具
func (r *Registry) Register(t tool.BaseTool) {
	r.tools = append(r.tools, t)
}

// GetAll 获取所有工具
func (r *Registry) GetAll() []tool.BaseTool {
	return r.tools
}

// InitTools 初始化所有内置工具
func InitTools() ([]tool.BaseTool, error) {
	registry := NewRegistry()

	ctx := context.Background()
	cm, _ := llm.NewChatModel(ctx)
	ragTool, err := BuildRAGTool(ctx, cm)
	if err != nil {
		return nil, fmt.Errorf("build rag tool: %w", err)
	}
	registry.Register(ragTool)

	//// 注册天气工具
	//weatherTool, err := NewWeatherTool()
	//if err != nil {
	//	return nil, fmt.Errorf("failed to create weather tool: %w", err)
	//}
	//registry.Register(weatherTool)
	//
	//// 注册计算器工具
	//calcTool, err := NewCalculatorTool()
	//if err != nil {
	//	return nil, fmt.Errorf("failed to create calculator tool: %w", err)
	//}
	//registry.Register(calcTool)

	// 注册搜索工具
	searchTool, err := NewSearchTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create search tool: %w", err)
	}
	registry.Register(searchTool)

	return registry.GetAll(), nil
}
