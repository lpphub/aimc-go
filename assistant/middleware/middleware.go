package middleware

import (
	"context"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/components/model"
)

// SetupMiddlewares 设置中间件
func SetupMiddlewares(ctx context.Context, summaryModel model.BaseChatModel) ([]adk.ChatModelAgentMiddleware, error) {
	var middlewares []adk.ChatModelAgentMiddleware

	// 1. 文件系统中间件（可选）
	fsBackend, err := local.NewBackend(ctx, &local.Config{})
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

	// 2. 总结中间件（压缩长对话历史）
	sumMW, err := summarization.New(ctx, &summarization.Config{
		Model: summaryModel,
		Trigger: &summarization.TriggerCondition{
			ContextTokens: 100000,
		},
	})
	if err != nil {
		return nil, err
	}
	middlewares = append(middlewares, sumMW)

	return middlewares, nil
}
