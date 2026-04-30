package runtime

import (
	"aimc-go/assistant/session"
	"aimc-go/assistant/store"
	"context"
	"fmt"

	adkstore "github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type RuntimeOption func(*Runtime)

func WithStore(s store.Store) RuntimeOption {
	return func(r *Runtime) {
		r.store = s
	}
}

func WithMaxRounds(n int) RuntimeOption {
	return func(r *Runtime) {
		r.maxRounds = n
	}
}

type Runtime struct {
	runner          *adk.Runner
	store           store.Store
	checkpointStore adk.CheckPointStore
	maxRounds       int
}

func New(agent adk.Agent, opts ...RuntimeOption) (*Runtime, error) {
	r := &Runtime{
		checkpointStore: adkstore.NewInMemoryStore(),
		maxRounds:       25,
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.store == nil {
		return nil, fmt.Errorf("store is required, use WithStore() to set")
	}

	r.runner = adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: r.checkpointStore,
	})

	return r, nil
}

func (r *Runtime) Generate(ctx context.Context, messages []*schema.Message, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	return r.runner.Run(ctx, messages, opts...)
}

func (r *Runtime) Events(ctx context.Context, messages []*schema.Message, opts ...adk.AgentRunOption) <-chan *adk.AgentEvent {
	out := make(chan *adk.AgentEvent, 32)

	go func() {
		defer close(out)
		iter := r.Generate(ctx, messages, opts...)

		for {
			if ctx.Err() != nil {
				return
			}
			event, ok := iter.Next()
			if !ok {
				return
			}
			out <- event
		}
	}()

	return out
}

func (r *Runtime) Run(ctx context.Context, sess *session.Session, query string, opts ...adk.AgentRunOption) error {
	if err := r.store.Append(ctx, sess.ID, schema.UserMessage(query)); err != nil {
		return fmt.Errorf("append user message: %w", err)
	}

	history, err := r.store.Get(ctx, sess.ID)
	if err != nil {
		return fmt.Errorf("get history: %w", err)
	}

	history = trimRounds(history, r.maxRounds)

	runOpts := append(opts, adk.WithCheckPointID(sess.ID))

	_ = sess.Emit(session.Event{Type: session.TypeMessage, Content: "🤖: "})
	iter := r.Generate(ctx, history, runOpts...)

	messages, interruptInfo, err := r.drain(iter, sess)
	if err != nil {
		return fmt.Errorf("process events: %w", err)
	}

	if len(messages) > 0 {
		if err = r.store.Append(ctx, sess.ID, messages...); err != nil {
			return fmt.Errorf("append messages: %w", err)
		}
	}

	if interruptInfo != nil {
		return r.handleInterrupt(ctx, sess, interruptInfo)
	}

	_ = sess.Emit(session.Event{Type: session.TypeMessage, Content: "\n"})

	return nil
}

func (r *Runtime) Resume(ctx context.Context, sess *session.Session, checkpointID string, resumeData map[string]any) (
	[]*schema.Message, *adk.InterruptInfo, error,
) {
	events, err := r.runner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{
		Targets: resumeData,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("resume with params: %w", err)
	}

	return r.drain(events, sess)
}

func trimRounds(history []*schema.Message, maxRounds int) []*schema.Message {
	if maxRounds <= 0 || len(history) == 0 {
		return history
	}

	keepStart := 0
	userCount := 0
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == schema.User {
			userCount++
			if userCount == maxRounds {
				keepStart = i
				break
			}
		}
	}

	if keepStart == 0 {
		return history
	}

	result := make([]*schema.Message, len(history)-keepStart)
	copy(result, history[keepStart:])
	return result
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}