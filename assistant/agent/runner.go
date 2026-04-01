package agent

import (
	"aimc-go/assistant/approval"
	"aimc-go/assistant/sink"
	"aimc-go/assistant/store"
	"context"
	"fmt"
	"strings"

	adkstore "github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

type Runner struct {
	inner    *adk.Runner
	handler  *EventHandler
	store    store.Store
	sink     sink.Sink
	approver approval.ApprovalHandler
}

type RunnerOption func(*Runner)

func WithStore(s store.Store) RunnerOption {
	return func(r *Runner) {
		r.store = s
	}
}

func WithSink(s sink.Sink) RunnerOption {
	return func(r *Runner) {
		r.sink = s
	}
}

func WithApprovalHandler(p approval.ApprovalHandler) RunnerOption {
	return func(r *Runner) {
		r.approver = p
	}
}

func NewRunner(agent adk.Agent, opts ...RunnerOption) (*Runner, error) {
	r := &Runner{
		inner: adk.NewRunner(context.Background(), adk.RunnerConfig{
			Agent:           agent,
			EnableStreaming: true,
			CheckPointStore: adkstore.NewInMemoryStore(), // 检查点，可替换redis实现
		}),
		handler: &EventHandler{},
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.store == nil {
		return nil, fmt.Errorf("store is required, use agent.JSONLStore(dir) for a JSONL store")
	}
	if r.sink == nil {
		return nil, fmt.Errorf("sink is required, use agent.StdoutSink() for stdout")
	}

	return r, nil
}

func (r *Runner) Run(ctx context.Context, sessionID, query string) error {
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	session, _ := r.store.GetOrCreate(ctx, sessionID)
	_ = r.store.Append(ctx, session.ID, schema.UserMessage(query))

	iter := r.inner.Run(ctx, session.Messages, adk.WithCheckPointID(sessionID))
	content, interruptInfo, err := r.processEventStream(ctx, iter)
	if err != nil {
		return err
	}

	for interruptInfo != nil {
		content, interruptInfo, err = r.handleInterrupt(ctx, sessionID, interruptInfo)
		if err != nil {
			return err
		}
	}

	_ = r.store.Append(ctx, session.ID, schema.AssistantMessage(content, nil))

	return nil
}

func (r *Runner) processEventStream(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent]) (string, *adk.InterruptInfo, error) {
	ec := &EventContext{
		Ctx:       ctx,
		Collector: &strings.Builder{},
		Sink:      r.sink,
	}

	ec.Sink.Output(sink.Event{Type: "message", Content: "🤖: "})
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		interruptInfo, err := r.handler.HandleEvent(ec, event)
		if err != nil {
			ec.Sink.Output(sink.Event{Type: "message", Content: event.Err.Error()})
			return "", nil, err
		}
		if interruptInfo != nil {
			return ec.Collector.String(), interruptInfo, nil
		}
	}

	return ec.Collector.String(), nil, nil
}

func (r *Runner) Resume(ctx context.Context, checkPointID string, resumeData map[string]any) (string, *adk.InterruptInfo, error) {
	events, err := r.inner.ResumeWithParams(ctx, checkPointID, &adk.ResumeParams{
		Targets: resumeData,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to resume: %w", err)
	}
	return r.processEventStream(ctx, events)
}

func (r *Runner) handleInterrupt(ctx context.Context, checkPointID string, interruptInfo *adk.InterruptInfo) (string, *adk.InterruptInfo, error) {
	for _, ic := range interruptInfo.InterruptContexts {
		if !ic.IsRootCause {
			continue
		}

		if r.approver == nil {
			return "", nil, fmt.Errorf("interrupt occurred but no approval handler configured")
		}

		result, err := r.approver.GetApproval(ctx, ic)
		if err != nil {
			return "", nil, fmt.Errorf("approval failed: %w", err)
		}

		content, newInterruptInfo, err := r.Resume(ctx, checkPointID, map[string]any{
			ic.ID: result,
		})
		if err != nil {
			return "", nil, err
		}

		return content, newInterruptInfo, nil
	}

	return "", nil, fmt.Errorf("no root cause interrupt context found")
}
