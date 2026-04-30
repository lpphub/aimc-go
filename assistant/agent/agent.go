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
	"github.com/cloudwego/eino/compose"
)

func New(ctx context.Context, opts ...Option) (adk.Agent, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	var err error

	if cfg.Model == nil {
		cfg.Model, err = llm.NewChatModel(ctx)
		if err != nil {
			return nil, fmt.Errorf("create model: %w", err)
		}
	}

	if cfg.Tools == nil {
		cfg.Tools, err = tools.InitTools(cfg.Model)
		if err != nil {
			return nil, fmt.Errorf("init tools: %w", err)
		}
	}

	if cfg.Middlewares == nil {
		cfg.Middlewares, err = middleware.SetupMiddlewares(ctx, cfg.Model, middleware.Config{
			SkillDir:    cfg.SkillDir,
			PlanTaskDir: cfg.PlanTaskDir,
		})
		if err != nil {
			return nil, fmt.Errorf("setup middlewares: %w", err)
		}
	}

	if cfg.Instruction == "" {
		if cfg.ProjectRoot != "" {
			cfg.Instruction = prompts.GetEinoAssistant(cfg.ProjectRoot)
		} else {
			cfg.Instruction = prompts.CodeAssistant
		}
	}

	return buildAgent(ctx, cfg)
}

func buildAgent(ctx context.Context, cfg *Config) (adk.Agent, error) {
	if cfg.Model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if len(cfg.Tools) == 0 {
		return nil, fmt.Errorf("tools is required")
	}
	if len(cfg.Middlewares) == 0 {
		return nil, fmt.Errorf("middlewares is required")
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
