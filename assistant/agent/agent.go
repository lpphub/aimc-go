package agent

import (
	"aimc-go/assistant/llm"
	"aimc-go/assistant/middleware"
	"aimc-go/assistant/tools"
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
)

type Config struct {
	Name          string
	Description   string
	Instruction   string
	MaxIterations int
}

func New(cfg Config) (adk.Agent, error) {
	ctx := context.Background()

	// llm模型
	cm, err := llm.NewChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("new chatModel: %w", err)
	}

	// 工具
	cfgTools, err := tools.InitTools()
	if err != nil {
		return nil, fmt.Errorf("init tools: %w", err)
	}

	// 中间件
	middlewares, err := middleware.SetupMiddlewares(ctx, cm)
	if err != nil {
		return nil, fmt.Errorf("setup middlewares: %w", err)
	}

	// 创建 Agent
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Instruction:   cfg.Instruction,
		MaxIterations: cfg.MaxIterations,
		Model:         cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfgTools,
			},
		},
		Handlers: middlewares,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agent, nil
}
