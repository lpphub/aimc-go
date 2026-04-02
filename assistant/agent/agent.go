package agent

import (
	"aimc-go/assistant/agent/llm"
	"aimc-go/assistant/agent/middleware"
	"aimc-go/assistant/agent/prompts"
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

func New(ctx context.Context) (adk.Agent, error) {
	// 1. model
	cm, err := llm.NewChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to create model: %v\n", err)
	}

	// 2. tools — 使用默认工具集
	agentTools, err := PresetTools(cm)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to initialize tools: %v\n", err)
	}

	// 3. middlewares — 使用默认中间件
	middlewares, err := PresetMiddlewares(ctx, cm, middleware.Config{
		SkillDir: "/home/lsk/projects/eino-demo/ext/skills",
	})
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to setup middlewares: %v\n", err)
	}

	// 4. agent
	ag, err := buildAgent(ctx, AgentConfig{
		Name:          "enio-assistant",
		Description:   "enio tutorial assistant",
		Instruction:   prompts.GetEinoAssistant("/home/lsk/projects/eino-demo"),
		Model:         cm,
		Tools:         agentTools,
		Middlewares:   middlewares,
		MaxIterations: 50,
	})
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to create agent: %v\n", err)
	}

	return ag, nil
}

func buildAgent(ctx context.Context, cfg AgentConfig) (adk.Agent, error) {
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
