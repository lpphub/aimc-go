package agent

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type Runner struct {
	runner   *adk.Runner
	pipeline *EventPipeline

	history []*schema.Message
}

// NewRunner 创建 Runner
func NewRunner(agent adk.Agent) *Runner {
	return &Runner{
		runner: adk.NewRunner(context.Background(), adk.RunnerConfig{
			Agent:           agent,
			EnableStreaming: true,
		}),
		pipeline: NewEventPipeline(
			&ErrorHandler{},
			&ActionHandler{},
			&ToolHandler{},
			&MessageHandler{},
		),
		history: make([]*schema.Message, 0),
	}
}

// Run 执行查询（便捷方法，用于单个用户消息）
func (r *Runner) Run(ctx context.Context, query string) (string, error) {
	// 添加用户消息
	r.history = append(r.history, schema.UserMessage(query))

	iter := r.runner.Run(ctx, r.history)

	content, err := r.processEventStream(ctx, iter)
	if err != nil {
		return "", err
	}

	// 添加助手消息
	r.history = append(r.history, schema.AssistantMessage(content, nil))

	return content, nil
}

func (r *Runner) processEventStream(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder

	ec := &EventContext{
		Ctx:       ctx,
		Collector: &sb,
		Writer:    &MsgWriter{},
	}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if err := r.pipeline.Execute(ec, event); err != nil {
			return "", err
		}
	}

	return sb.String(), nil
}
