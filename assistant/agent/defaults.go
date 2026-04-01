package agent

import (
	"aimc-go/assistant/agent/middleware"
	"aimc-go/assistant/agent/tools"
	"aimc-go/assistant/sink"
	"aimc-go/assistant/store"
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

// DefaultTools returns the built-in tools (RAG + search).
func DefaultTools(cm model.BaseChatModel) ([]tool.BaseTool, error) {
	return tools.InitTools(cm)
}

// DefaultMiddlewares returns the built-in infra middlewares.
func DefaultMiddlewares(ctx context.Context, cm model.BaseChatModel, cfg middleware.Config) ([]adk.ChatModelAgentMiddleware, error) {
	return middleware.SetupMiddlewares(ctx, cm, cfg)
}

// DefaultStore returns a JSONL store at the given directory.
func DefaultStore(dir string) store.Store {
	return &store.JSONLStore{
		Dir:   dir,
		Cache: make(map[string]*store.Session),
	}
}

// DefaultSink returns a stdout sink.
func DefaultSink() sink.Sink {
	return &sink.StdoutSink{}
}
