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
	sink    sink.Sink
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

func NewRunner(agent adk.Agent, opts ...RunnerOption) *Runner {
	r := &Runner{
		inner: adk.NewRunner(context.Background(), adk.RunnerConfig{
			Agent:           agent,
			EnableStreaming: true,
		}),
		handler: &EventHandler{},
		store: &store.JSONLStore{
			Dir:   "./data/sessions",
			Cache: make(map[string]*store.Session),
		},
		sink: &sink.StdoutSink{},
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *Runner) Run(ctx context.Context, sessionID, query string) (string, error) {
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	session, _ := r.store.GetOrCreate(ctx, sessionID)
	_ = r.store.Append(ctx, session.ID, schema.UserMessage(query))

	history := session.Messages
	iter := r.inner.Run(ctx, history)
	content, err := r.processEventStream(ctx, iter)
	if err != nil {
		return "", err
	}

	_ = r.store.Append(ctx, session.ID, schema.AssistantMessage(content, nil))
	return content, nil
}

func (r *Runner) processEventStream(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	ec := &EventContext{
		Ctx:       ctx,
		Collector: &strings.Builder{},
		Sink:      r.sink,
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
