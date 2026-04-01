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
	Name          string
	Description   string
	Instruction   string
	MaxIterations int // 0 defaults to 30

	Model       model.ToolCallingChatModel     // required
	Tools       []tool.BaseTool                // required
	Middlewares []adk.ChatModelAgentMiddleware // required
}

// DefaultTools returns the built-in tools (RAG + search).
func DefaultTools(cm model.BaseChatModel) ([]tool.BaseTool, error) {
	return tools.InitTools(cm)
}

// DefaultMiddlewares returns the built-in infra middlewares.
func DefaultMiddlewares(ctx context.Context, cm model.BaseChatModel, cfg middleware.Config) ([]adk.ChatModelAgentMiddleware, error) {
	return middleware.SetupMiddlewares(ctx, cm, cfg)
}

func New(ctx context.Context, cfg AgentConfig) (adk.Agent, error) {
	if cfg.Model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if len(cfg.Tools) == 0 {
		return nil, fmt.Errorf("tools is required, use agent.DefaultTools(cm) for built-in tools")
	}
	if len(cfg.Middlewares) == 0 {
		return nil, fmt.Errorf("middlewares is required, use agent.DefaultMiddlewares(ctx, cm, middleware.Config{}) for built-in middlewares")
	}
	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 30
	}

	ag, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Instruction:   cfg.Instruction,
		MaxIterations: cfg.MaxIterations,
		Model:         cfg.Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.Tools,
			},
		},
		Handlers: cfg.Middlewares,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return ag, nil
}
