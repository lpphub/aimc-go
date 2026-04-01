package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

type AgentConfig struct {
	Name          string
	Description   string
	Instruction   string
	MaxIterations int // 0 defaults to 50

	Model       model.ToolCallingChatModel     // required
	Tools       []tool.BaseTool                // required
	Middlewares []adk.ChatModelAgentMiddleware // required
}

func New(ctx context.Context, cfg AgentConfig) (adk.Agent, error) {
	if cfg.Model == nil {
		return nil, fmt.Errorf("model is required")
	}

	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 50
	}

	if len(cfg.Tools) == 0 {
		return nil, fmt.Errorf("tools is required, use agent.PresetTools(cm) for built-in tools")
	}

	if len(cfg.Middlewares) == 0 {
		return nil, fmt.Errorf("middlewares is required, use agent.PresetMiddlewares(ctx, cm, middleware.Config{}) for built-in middlewares")
	}

	ag, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Instruction:   cfg.Instruction,
		MaxIterations: cfg.MaxIterations,
		Model:         cfg.Model,
		Handlers:      cfg.Middlewares,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.Tools,
			},
		},
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries: 3,
			IsRetryAble: func(ctx context.Context, err error) bool {
				// 429 限流错误可重试
				return strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "qpm limit")
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return ag, nil
}
