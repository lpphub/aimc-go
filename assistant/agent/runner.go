package agent

import (
	"aimc-go/assistant/sink"
	"aimc-go/assistant/store"
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

type Runner struct {
	inner   *adk.Runner
	handler *EventHandler
	store   store.Store
}

// NewRunner 创建 Runner
func NewRunner(agent adk.Agent) *Runner {
	return &Runner{
		inner: adk.NewRunner(context.Background(), adk.RunnerConfig{
			Agent:           agent,
			EnableStreaming: true,
		}),
		handler: &EventHandler{},
		store: &store.JSONLStore{
			Dir:   "./data/sessions",
			Cache: make(map[string]*store.Session),
		},
	}
}

// Run 执行查询（便捷方法，用于单个用户消息）
func (r *Runner) Run(ctx context.Context, sessionID, query string) (string, error) {
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	session, _ := r.store.GetOrCreate(ctx, sessionID)

	// 添加用户消息
	_ = r.store.Append(ctx, session.ID, schema.UserMessage(query))

	history := session.Messages

	iter := r.inner.Run(ctx, history)
	content, err := r.processEventStream(ctx, iter)
	if err != nil {
		return "", err
	}

	// 添加助手消息（未记录工具调用）
	_ = r.store.Append(ctx, session.ID, schema.AssistantMessage(content, nil))

	return content, nil
}

func (r *Runner) processEventStream(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	ec := &EventContext{
		Ctx:       ctx,
		Collector: &strings.Builder{},
		Sink:      &sink.StdoutSink{},
	}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		err := r.handler.HandleEvent(ec, event)
		if err != nil {
			return "", err
		}
	}

	return ec.Collector.String(), nil
}
