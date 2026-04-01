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

// Config middleware configuration
type Config struct {
	SkillDir string // skill files directory, empty means no skill middleware
}

func SetupMiddlewares(ctx context.Context, chatModel model.BaseChatModel, cfg Config) ([]adk.ChatModelAgentMiddleware, error) {
	middlewares, err := setupInfraMiddleware(ctx, chatModel, cfg)
	if err != nil {
		return nil, err
	}

	middlewares = append(middlewares, &safeToolMiddleware{})

	return middlewares, nil
}

func setupInfraMiddleware(ctx context.Context, chatModel model.BaseChatModel, cfg Config) ([]adk.ChatModelAgentMiddleware, error) {
	var middlewares []adk.ChatModelAgentMiddleware

	backend, err := local.NewBackend(ctx, &local.Config{})
	if err != nil {
		return nil, err
	}

	if patchMW, err := patch(ctx); err == nil {
		middlewares = append(middlewares, patchMW)
	}

	if sumMW, err := sum(ctx, chatModel); err == nil {
		middlewares = append(middlewares, sumMW)
	}

	if reductionMW, err := reduce(ctx, backend); err == nil {
		middlewares = append(middlewares, reductionMW)
	}

	if fsMW, err := fs(ctx, backend); err == nil {
		middlewares = append(middlewares, fsMW)
	}

	if cfg.SkillDir != "" {
		if skillMW, err := skills(ctx, backend, cfg.SkillDir); err == nil {
			middlewares = append(middlewares, skillMW)
		}
	}

	return middlewares, nil
}

// fs injects file operation tools
func fs(ctx context.Context, backend *local.Local) (adk.ChatModelAgentMiddleware, error) {
	fsMW, err := filesystem.New(ctx, &filesystem.MiddlewareConfig{
		Backend:        backend,
		StreamingShell: backend,
	})
	if err != nil {
		return nil, err
	}
	return fsMW, nil
}

func skills(ctx context.Context, backend *local.Local, skillDir string) (adk.ChatModelAgentMiddleware, error) {
	skillBackend, _ := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
		Backend: backend,
		BaseDir: skillDir,
	})
	skillMW, err := skill.NewMiddleware(ctx, &skill.Config{
		Backend: skillBackend,
	})
	if err != nil {
		return nil, err
	}
	return skillMW, nil
}

func patch(ctx context.Context) (adk.ChatModelAgentMiddleware, error) {
	patchMW, err := patchtoolcalls.New(ctx, &patchtoolcalls.Config{})
	if err != nil {
		return nil, err
	}
	return patchMW, nil
}

func reduce(ctx context.Context, backend *local.Local) (adk.ChatModelAgentMiddleware, error) {
	reductionMW, err := reduction.New(ctx, &reduction.Config{Backend: backend})
	if err != nil {
		return nil, err
	}
	return reductionMW, nil
}

func sum(ctx context.Context, summaryModel model.BaseChatModel) (adk.ChatModelAgentMiddleware, error) {
	sumMW, err := summarization.New(ctx, &summarization.Config{
		Model: summaryModel,
		Trigger: &summarization.TriggerCondition{
			ContextTokens: 100000,
		},
	})
	if err != nil {
		return nil, err
	}
	return sumMW, nil
}
