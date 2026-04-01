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

// SetupMiddlewares 设置中间件（注入基础设施能力）
//
//	Handlers: []adk.ChatModelAgentMiddleware{
//	       patchToolCallsMW,    // ← 必须放在第一个！
//	       summarizationMW,     // 2. 压缩上下文
//	       reductionMW,         // 3. 处理大型工具结果
//	       filesystemMW,        // 4. 添加文件工具
//	       skillMW,             // 5. 添加技能发现
//	       planTaskMW,          // 6. 添加任务管理
//	   },
func SetupMiddlewares(ctx context.Context, chatModel model.BaseChatModel) ([]adk.ChatModelAgentMiddleware, error) {
	middlewares, err := setupInfraMiddleware(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	// 自定义中间件
	middlewares = append(middlewares, &safeToolMiddleware{})

	return middlewares, nil
}

func setupInfraMiddleware(ctx context.Context, chatModel model.BaseChatModel) ([]adk.ChatModelAgentMiddleware, error) {
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

	if skillMW, err := skills(ctx, backend); err == nil {
		middlewares = append(middlewares, skillMW)
	}
	return middlewares, nil
}

// 注入文件操作相关工具
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

func skills(ctx context.Context, backend *local.Local) (adk.ChatModelAgentMiddleware, error) {
	skillBackend, _ := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
		Backend: backend,
		BaseDir: "/home/lsk/projects/eino-demo/ext/skills",
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
