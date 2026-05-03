package agent

import (
	"context"
	"strings"

	"aimc-go/assistant/agent/middlewares"
	agentTools "aimc-go/assistant/agent/tools"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func NewChatAgent(ctx context.Context, opts ...Option) (adk.Agent, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	d, err := resolveDeps(ctx, cfg)
	if err != nil {
		return nil, err
	}

	toolList, err := buildTools(ctx, d.Model)
	if err != nil {
		return nil, err
	}

	mw, err := middlewares.NewChainBuilder(ctx, d.Model, d.Backend).
		WithPatch().
		WithSummarization(850000).
		WithReduction().
		WithFilesystem().
		WithSkill(cfg.SkillDir).
		WithPlanTask(cfg.PlanTaskDir).
		WithToolRecovery().
		Build()
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:             cfg.Name,
		Description:      cfg.Description,
		Instruction:      resolveInstruction(cfg),
		MaxIterations:    cfg.MaxIterations,
		Model:            d.Model,
		Handlers:         mw,
		ToolsConfig:      toolsConfig(toolList),
		ModelRetryConfig: retryConfig(),
	})
}

func NewDeepAgent(ctx context.Context, opts ...Option) (adk.Agent, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	d, err := resolveDeps(ctx, cfg)
	if err != nil {
		return nil, err
	}

	toolList, err := buildTools(ctx, d.Model)
	if err != nil {
		return nil, err
	}

	mw, err := middlewares.NewChainBuilder(ctx, d.Model, d.Backend).
		WithPatch().
		WithSummarization(850000).
		WithReduction().
		WithSkill(cfg.SkillDir).
		WithToolRecovery().
		Build()
	if err != nil {
		return nil, err
	}

	subAgent, _ := NewChatAgent(ctx, opts...)

	return deep.New(ctx, &deep.Config{
		Name:                   cfg.Name,
		Description:            cfg.Description,
		ChatModel:              d.Model,
		Instruction:            resolveInstruction(cfg),
		MaxIteration:           cfg.MaxIterations,
		Backend:                d.Backend,
		StreamingShell:         d.Backend,
		WithoutGeneralSubAgent: true, // 禁止通用子agent（避免递归委派）
		SubAgents:              []adk.Agent{subAgent},
		ToolsConfig:            toolsConfig(toolList),
		Handlers:               mw,
		ModelRetryConfig:       retryConfig(),
	})
}

func buildTools(ctx context.Context, m model.ToolCallingChatModel) ([]tool.BaseTool, error) {
	return agentTools.NewChainBuilder(ctx, m).
		WithRAG().
		WithTime().
		Build()
}

func toolsConfig(tools []tool.BaseTool) adk.ToolsConfig {
	return adk.ToolsConfig{
		ToolsNodeConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
	}
}

func retryConfig() *adk.ModelRetryConfig {
	return &adk.ModelRetryConfig{
		MaxRetries: 3,
		IsRetryAble: func(_ context.Context, err error) bool {
			s := err.Error()
			return strings.Contains(s, "429") || strings.Contains(s, "qpm limit")
		},
	}
}
