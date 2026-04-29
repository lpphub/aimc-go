package callback

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

//
// =========================
// UsageStats（global / external injected）
// =========================
//

type usageStatsKey struct{}

type UsageStats struct {
	CallCounts       atomic.Int64
	PromptTokens     atomic.Int64
	CompletionTokens atomic.Int64
	TotalTokens      atomic.Int64
	DurationMs       atomic.Int64
}

func (s *UsageStats) Add(usage *model.TokenUsage) {
	if usage == nil {
		return
	}
	s.PromptTokens.Add(int64(usage.PromptTokens))
	s.CompletionTokens.Add(int64(usage.CompletionTokens))
	s.TotalTokens.Add(int64(usage.TotalTokens))
}

func (s *UsageStats) AddDuration(d time.Duration) {
	s.DurationMs.Add(d.Milliseconds())
}

func (s *UsageStats) Report() {
	fmt.Printf(
		"\n 📊 Usage：Call Counts: %d, Prompt Tokens: %d, Completion Tokens: %d, Total Tokens: %d, Duration(ms): %d \n",
		s.CallCounts.Load(),
		s.PromptTokens.Load(),
		s.CompletionTokens.Load(),
		s.TotalTokens.Load(),
		s.DurationMs.Load(),
	)
}

//
// =========================
// ctx helpers（强烈推荐）
// =========================
//

func WithUsageStats(ctx context.Context, stats *UsageStats) context.Context {
	return context.WithValue(ctx, usageStatsKey{}, stats)
}

func getStats(ctx context.Context) (*UsageStats, bool) {
	v, ok := ctx.Value(usageStatsKey{}).(*UsageStats)
	return v, ok
}

//
// =========================
// usageState（per-call）
// =========================
//

type usageStateKey struct{}

type usageState struct {
	start     time.Time
	msgCounts int
}

func NewUsageHandler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if info.Component != components.ComponentOfChatModel {
				return ctx
			}

			// stats（必须外部注入）
			if stats, ok := getStats(ctx); ok {
				stats.CallCounts.Add(1)
			}

			in := model.ConvCallbackInput(input)
			if in == nil {
				return ctx
			}

			// ctx state
			ctx = context.WithValue(ctx, usageStateKey{}, &usageState{
				start:     time.Now(),
				msgCounts: len(in.Messages),
			})

			return ctx
		}).
		OnEndWithStreamOutputFn(func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
			if info.Component != components.ComponentOfChatModel {
				return ctx
			}

			go func() {
				defer output.Close()

				stats, ok := getStats(ctx)
				if !ok {
					return
				}

				var usage *model.TokenUsage
				for {
					chunk, err := output.Recv()
					if err != nil {
						break
					}

					cbOutput := model.ConvCallbackOutput(chunk)
					if cbOutput == nil {
						continue
					}

					if cbOutput.TokenUsage != nil {
						usage = cbOutput.TokenUsage
					}
				}

				stats.Add(usage)

				if state, ok := ctx.Value(usageStateKey{}).(*usageState); ok {
					stats.AddDuration(time.Since(state.start))
				}
			}()

			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if info.Component == components.ComponentOfChatModel {
				fmt.Printf("[ChatModel Error] %v\n", err)
			}
			return ctx
		}).
		Build()
}
