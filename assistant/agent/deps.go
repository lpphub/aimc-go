package agent

import (
	"aimc-go/assistant/agent/prompt"
	"aimc-go/assistant/agent/provider"
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/components/model"
)

type Deps struct {
	Model   model.ToolCallingChatModel
	Backend *local.Local
}

func resolveDeps(ctx context.Context, cfg *Config) (*Deps, error) {
	d := &Deps{}

	if cfg.Model != nil {
		d.Model = cfg.Model
	} else {
		m, err := provider.NewProviderFromEnv().NewChatModel(ctx)
		if err != nil {
			return nil, fmt.Errorf("create model: %w", err)
		}
		d.Model = m
	}

	backend, err := local.NewBackend(ctx, &local.Config{})
	if err != nil {
		return nil, fmt.Errorf("create backend: %w", err)
	}
	d.Backend = backend

	return d, nil
}

func resolveInstruction(cfg *Config) string {
	if cfg.Instruction != "" {
		return cfg.Instruction
	}
	if cfg.ProjectRoot != "" {
		return prompt.GetEinoAssistant(cfg.ProjectRoot)
	}
	return prompt.CodeAssistant
}