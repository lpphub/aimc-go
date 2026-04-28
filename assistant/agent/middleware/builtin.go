package middleware

import (
	"context"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/patchtoolcalls"
	"github.com/cloudwego/eino/adk/middlewares/plantask"
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
			ContextTokens: 850000,
		},
		//PreserveUserMessages: &summarization.PreserveUserMessages{
		//	Enabled:   true,
		//	MaxTokens: 50000,
		//},
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

	// plantask
	if cfg.PlanTaskDir != "" {
		ptw, err := plantask.New(ctx, &plantask.Config{
			Backend: &planTaskBackend{Local: backend}, // 必需：存储后端
			BaseDir: cfg.PlanTaskDir,                  // 必需：任务文件目录
		})
		if err != nil {
			return nil, err
		}
		middlewares = append(middlewares, ptw)
	}

	return middlewares, nil
}

type planTaskBackend struct {
	*local.Local
}

func (b *planTaskBackend) Delete(ctx context.Context, req *plantask.DeleteRequest) error {
	return nil
}
