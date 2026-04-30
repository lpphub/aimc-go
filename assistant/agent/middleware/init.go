package middleware

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type Config struct {
	SkillDir    string
	PlanTaskDir string
}

func SetupMiddlewares(ctx context.Context, chatModel model.BaseChatModel, cfg Config) ([]adk.ChatModelAgentMiddleware, error) {
	middlewares, err := setupBuiltInMiddleware(ctx, chatModel, cfg)
	if err != nil {
		return nil, err
	}

	//middlewares = append(middlewares, NewApprovalMiddleware("execute"))
	middlewares = append(middlewares, NewToolRecoveryMiddleware())

	return middlewares, nil
}

func singleChunkReader(msg string) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r
}
