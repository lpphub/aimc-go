package middleware

import (
	"context"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/components/model"
)

// SetupMiddlewares 设置中间件
func SetupMiddlewares(ctx context.Context, summaryModel model.BaseChatModel) ([]adk.ChatModelAgentMiddleware, error) {
	var middlewares []adk.ChatModelAgentMiddleware

	// 1. 文件系统中间件（可选）
	if cfg.Middleware.Filesystem.Enabled {
		fsBackend, err := local.NewBackend(ctx, &local.Config{
			RootPath: cfg.Middleware.Filesystem.BaseDir,
		})
		if err != nil {
			return nil, err
		}

		fsMW, err := filesystem.New(ctx, &filesystem.MiddlewareConfig{
			Backend: fsBackend,
		})
		if err != nil {
			return nil, err
		}
		middlewares = append(middlewares, fsMW)
	}

	// 2. 总结中间件（压缩长对话历史）
	if cfg.Middleware.Summarization.Enabled && summaryModel != nil {
		sumMW, err := summarization.New(ctx, &summarization.Config{
			Model: summaryModel,
			Trigger: &summarization.TriggerCondition{
				ContextTokens: cfg.Middleware.Summarization.ContextTokens,
			},
		})
		if err != nil {
			return nil, err
		}
		middlewares = append(middlewares, sumMW)
	}

	// 3. 缩减中间件（处理大型工具结果）
	if cfg.Middleware.Reduction.Enabled {
		// 需要一个 backend 来存储被裁剪的内容
		reductionBackend, err := local.NewBackend(ctx, &local.Config{
			RootPath: "./.reduction_cache",
		})
		if err != nil {
			return nil, err
		}

		redMW, err := reduction.New(ctx, &reduction.Config{
			Backend:           reductionBackend,
			MaxTokensForClear: cfg.Middleware.Reduction.MaxTokens,
		})
		if err != nil {
			return nil, err
		}
		middlewares = append(middlewares, redMW)
	}

	return middlewares, nil
}
