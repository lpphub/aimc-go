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

// Validate 验证 AgentConfig 配置
func (c *AgentConfig) Validate() error {
	if c.Model == nil {
		return fmt.Errorf("model is required")
	}
	if len(c.Tools) == 0 {
		return fmt.Errorf("tools is required, use agent.PresetTools(cm) for built-in tools")
	}
	if len(c.Middlewares) == 0 {
		return fmt.Errorf("middlewares is required, use agent.PresetMiddlewares(ctx, cm, middleware.Config{}) for built-in middlewares")
	}
	return nil
}

func New(ctx context.Context, cfg AgentConfig) (adk.Agent, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 50
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
