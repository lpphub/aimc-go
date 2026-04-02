package agent

import (
	"aimc-go/assistant/agent/llm"
	"aimc-go/assistant/agent/middleware"
	"aimc-go/assistant/agent/prompts"
	"aimc-go/assistant/agent/tools"
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// New 创建默认 Agent
func New(ctx context.Context, opts ...Option) (adk.Agent, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// 1. model
	cm, err := llm.NewChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("create model: %w", err)
	}

	// 2. tools
	agentTools := cfg.Tools
	if agentTools == nil {
		agentTools, err = tools.InitTools(cm)
		if err != nil {
			return nil, fmt.Errorf("init tools: %w", err)
		}
	}

	// 3. middlewares
	middlewares := cfg.Middlewares
	if middlewares == nil {
		middlewares, err = middleware.SetupMiddlewares(ctx, cm, middleware.Config{
			SkillDir: cfg.SkillDir,
		})
		if err != nil {
			return nil, fmt.Errorf("setup middlewares: %w", err)
		}
	}

	// 4. instruction
	instruction := prompts.CodeAssistant
	if cfg.ProjectRoot != "" {
		instruction = prompts.GetEinoAssistant(cfg.ProjectRoot)
	}

	// 5. build agent
	return buildAgent(ctx, buildConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Instruction:   instruction,
		MaxIterations: cfg.MaxIterations,
		Model:         cm,
		Tools:         agentTools,
		Middlewares:   middlewares,
	})
}

// buildConfig 构建参数（内部使用）
type buildConfig struct {
	Name          string
	Description   string
	Instruction   string
	MaxIterations int

	Model       model.ToolCallingChatModel
	Tools       []tool.BaseTool
	Middlewares []adk.ChatModelAgentMiddleware
}

func buildAgent(ctx context.Context, cfg buildConfig) (adk.Agent, error) {
	if cfg.Model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if len(cfg.Tools) == 0 {
		return nil, fmt.Errorf("tools is required")
	}
	if len(cfg.Middlewares) == 0 {
		return nil, fmt.Errorf("middlewares is required")
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
				return strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "qpm limit")
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	return ag, nil
}
