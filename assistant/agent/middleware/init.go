package middleware

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Config middleware 配置
type Config struct {
	// SkillDir skill 文件目录，为空则不启用 skill 中间件
	SkillDir    string
	PlanTaskDir string
}

func SetupMiddlewares(ctx context.Context, chatModel model.BaseChatModel, cfg Config) ([]adk.ChatModelAgentMiddleware, error) {
	middlewares, err := setupBuiltInMiddleware(ctx, chatModel, cfg)
	if err != nil {
		return nil, err
	}

	middlewares = append(middlewares, NewApprovalMiddleware("execute"))
	middlewares = append(middlewares, &safeToolMiddleware{})

	return middlewares, nil
}

func singleChunkReader(msg string) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r
}
