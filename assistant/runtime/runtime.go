package runtime

import (
	"aimc-go/assistant/approval"
	"aimc-go/assistant/channel"
	"aimc-go/assistant/store"
	"context"
	"fmt"

	adkstore "github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// Runtime 服务层核心，管理完整的对话生命周期
type Runtime struct {
	runner          *adk.Runner
	store           store.Store
	checkpointStore adk.CheckPointStore
	handler         EventHandler // 事件处理器（无状态）
}

// RuntimeOption Runtime 配置选项
type RuntimeOption func(*Runtime)

// WithStore 设置存储
func WithStore(s store.Store) RuntimeOption {
	return func(r *Runtime) {
		r.store = s
	}
}

// New 创建 Runtime
func New(agent adk.Agent, opts ...RuntimeOption) (*Runtime, error) {
	r := &Runtime{
		handler:         EventHandler{},
		checkpointStore: adkstore.NewInMemoryStore(), // 默认内存 checkpoint
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.store == nil {
		return nil, fmt.Errorf("store is required, use WithStore() to set")
	}

	// 创建 Runner
	r.runner = adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: r.checkpointStore,
	})

	return r, nil
}

// Generate 执行对话，返回事件迭代器（原始 API）
// 用法：
//
//	iter := rt.Generate(ctx, messages, checkpointID)
//	for {
//	    event, ok := iter.Next()
//	    if !ok { break }
//	    // 处理 event.Err / event.Action / event.Output
//	}
func (r *Runtime) Generate(ctx context.Context, messages []*schema.Message, checkpointID string) *adk.AsyncIterator[*adk.AgentEvent] {
	return r.runner.Run(ctx, messages, adk.WithCheckPointID(checkpointID))
}

// Events 执行对话，返回事件 channel（便利 API）
// 用法：
//
//	for event := range rt.Events(ctx, messages, checkpointID) {
//	    if event.Err != nil { ... }
//	    if event.Output != nil { ... }
//	}
func (r *Runtime) Events(ctx context.Context, messages []*schema.Message, checkpointID string) <-chan *adk.AgentEvent {
	out := make(chan *adk.AgentEvent, 32)

	go func() {
		defer close(out)
		iter := r.Generate(ctx, messages, checkpointID)

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

// Run 执行一轮对话（完整流程）
func (r *Runtime) Run(ctx context.Context, ch *channel.Channel, query string) error {
	// 1. 存储用户消息
	if err := r.store.Append(ctx, ch.ID, schema.UserMessage(query)); err != nil {
		return fmt.Errorf("append user message: %w", err)
	}

	// 2. 获取历史消息
	sessHistory, err := r.store.GetOrCreate(ctx, ch.ID)
	if err != nil {
		return fmt.Errorf("get session history: %w", err)
	}

	// 3. 运行 agent
	_ = ch.Write(channel.Chunk{Type: channel.TypeMessage, Content: "🤖: "})
	iter := r.Generate(ctx, sessHistory.Messages, ch.ID)

	// 4. 处理事件流
	messages, interruptInfo, err := r.handler.Drain(iter, ch)
	if err != nil {
		return fmt.Errorf("process events: %w", err)
	}

	// 5. 存储输出消息
	if len(messages) > 0 {
		if err = r.store.Append(ctx, ch.ID, messages...); err != nil {
			return fmt.Errorf("append messages: %w", err)
		}
	}

	// 6. 处理中断（审批）
	if interruptInfo != nil {
		return r.handleInterrupt(ctx, ch, interruptInfo)
	}

	// 7. 发送完成信号
	_ = ch.Write(channel.Chunk{Type: channel.TypeDone})

	return nil
}

// Resume 恢复中断的对话
func (r *Runtime) Resume(ctx context.Context, ch *channel.Channel, checkpointID string, resumeData map[string]any) (
	[]*schema.Message, *adk.InterruptInfo, error,
) {
	events, err := r.runner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{
		Targets: resumeData,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("resume with params: %w", err)
	}

	return r.handler.Drain(events, ch)
}

// handleInterrupt 处理中断（审批）
func (r *Runtime) handleInterrupt(ctx context.Context, ch *channel.Channel, interruptInfo *adk.InterruptInfo) error {
	for _, ic := range interruptInfo.InterruptContexts {
		if !ic.IsRootCause {
			continue
		}

		// 发送审批请求
		approvalID := ic.ID
		info, ok := ic.Info.(*approval.Info)
		if !ok {
			return fmt.Errorf("unexpected interrupt info type: %T", ic.Info)
		}

		_ = ch.Write(channel.Chunk{
			Type:    channel.TypeApproval,
			Content: info.String(),
			Meta:    map[string]any{"approval_id": approvalID, "tool_name": info.ToolName},
		})

		// 等待审批结果
		input, err := ch.WaitInput(ctx)
		if err != nil {
			return fmt.Errorf("wait approval input: %w", err)
		}

		if input.Type != channel.InputApproval {
			return fmt.Errorf("unexpected input type: %s", input.Type)
		}

		result, ok := input.Data.(*approval.Result)
		if !ok {
			return fmt.Errorf("unexpected approval result type: %T", input.Data)
		}

		// 校验 ApprovalID（SSE 场景需要匹配，CLI 场景可跳过）
		if result.ApprovalID != "" && result.ApprovalID != approvalID {
			return fmt.Errorf("approval ID mismatch: expected %s, got %s", approvalID, result.ApprovalID)
		}

		// 发送审批结果通知
		if result.Approved {
			_ = ch.Write(channel.Chunk{Type: channel.TypeApprovalRes, Content: "✔️ Approved, executing...\n"})
		} else {
			_ = ch.Write(channel.Chunk{Type: channel.TypeApprovalRes, Content: "✖️ Rejected\n"})
		}

		// 恢复执行
		messages, newInterrupt, err := r.Resume(ctx, ch, ch.ID, map[string]any{
			approvalID: result,
		})
		if err != nil {
			return fmt.Errorf("resume after approval: %w", err)
		}

		// 存储恢复后的消息
		if len(messages) > 0 {
			if err = r.store.Append(ctx, ch.ID, messages...); err != nil {
				return fmt.Errorf("append resumed messages: %w", err)
			}
		}

		// 递归处理后续中断
		if newInterrupt != nil {
			return r.handleInterrupt(ctx, ch, newInterrupt)
		}
	}

	// 发送完成信号
	_ = ch.Write(channel.Chunk{Type: channel.TypeDone})

	return nil
}
