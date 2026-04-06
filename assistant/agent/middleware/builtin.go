package middleware

import (
	"context"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/patchtoolcalls"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/components/model"
)

// setupBuiltInMiddleware 初始化内置中间件
func setupBuiltInMiddleware(ctx context.Context, chatModel model.BaseChatModel, cfg Config) ([]adk.ChatModelAgentMiddleware, error) {
	backend, err := local.NewBackend(ctx, &local.Config{})
	if err != nil {
		return nil, err
	}

	var middlewares []adk.ChatModelAgentMiddleware

	// patch tool calls
	patchMW, err := patchtoolcalls.New(ctx, &patchtoolcalls.Config{})
	if err != nil {
		return nil, err
	}
	middlewares = append(middlewares, patchMW)

	// summarization
	sumMW, err := summarization.New(ctx, &summarization.Config{
		Model: chatModel,
		Trigger: &summarization.TriggerCondition{
			ContextTokens: 100000,
		},
	})
	if err != nil {
		return nil, err
	}
	middlewares = append(middlewares, sumMW)

	// reduction
	reductionMW, err := reduction.New(ctx, &reduction.Config{Backend: backend})
	if err != nil {
		return nil, err
	}
	middlewares = append(middlewares, reductionMW)

	// filesystem
	fsMW, err := filesystem.New(ctx, &filesystem.MiddlewareConfig{
		Backend:        backend,
		StreamingShell: backend,
	})
	if err != nil {
		return nil, err
	}
	middlewares = append(middlewares, fsMW)

	// skills (可选)
	if cfg.SkillDir != "" {
		skillBackend, _ := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
			Backend: backend,
			BaseDir: cfg.SkillDir,
		})
		skillMW, err := skill.NewMiddleware(ctx, &skill.Config{
			Backend: skillBackend,
		})
		if err != nil {
			return nil, err
		}
		middlewares = append(middlewares, skillMW)
	}

	return middlewares, nil
}
