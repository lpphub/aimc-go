package agent

import (
	"aimc-go/assistant/agent/middleware"
	"aimc-go/assistant/agent/tools"
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

type AgentConfig struct {
	// Metadata
	Name        string
	Description string
	Instruction string

	// Dependencies (injected by caller)
	Model       model.ToolCallingChatModel     // required
	Tools       []tool.BaseTool                // optional, nil = use defaults
	Middlewares []adk.ChatModelAgentMiddleware // optional, nil = use defaults

	// Runtime config
	MaxIterations int
}

func New(ctx context.Context, cfg AgentConfig) (adk.Agent, error) {
	if cfg.Model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 30
	}

	// If tools not specified, use defaults (requires model)
	if cfg.Tools == nil {
		defaultTools, err := tools.InitTools(cfg.Model)
		if err != nil {
			return nil, fmt.Errorf("init default tools: %w", err)
		}
		cfg.Tools = defaultTools
	}

	// If middlewares not specified, use default infra middlewares
	if cfg.Middlewares == nil {
		defaultMW, err := middleware.SetupMiddlewares(ctx, cfg.Model, middleware.Config{})
		if err != nil {
			return nil, fmt.Errorf("setup default middlewares: %w", err)
		}
		cfg.Middlewares = defaultMW
	}

	var toolsConfig adk.ToolsConfig
	if len(cfg.Tools) > 0 {
		toolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.Tools,
			},
		}
	}

	ag, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Instruction:   cfg.Instruction,
		MaxIterations: cfg.MaxIterations,
		Model:         cfg.Model,
		ToolsConfig:   toolsConfig,
		Handlers:      cfg.Middlewares,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return ag, nil
}
