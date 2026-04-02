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

// PresetTools returns the built-in tools (RAG + search).
func PresetTools(cm model.BaseChatModel) ([]tool.BaseTool, error) {
	return tools.InitTools(cm)
}

// PresetMiddlewares returns the built-in infra middlewares.
func PresetMiddlewares(ctx context.Context, cm model.BaseChatModel, cfg middleware.Config) ([]adk.ChatModelAgentMiddleware, error) {
	return middleware.SetupMiddlewares(ctx, cm, cfg)
}

// JSONLStore returns a JSONL store at the given directory.
func JSONLStore(dir string) store.Store {
	return &store.JSONLStore{
		Dir:   dir,
		Cache: make(map[string]*store.Session),
	}
}

// StdoutSink returns a stdout sink.
func StdoutSink() sink.Sink {
	return sink.NewStdoutSink()
}
