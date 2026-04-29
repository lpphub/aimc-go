package runtime

import (
	"aimc-go/assistant/session"
	"aimc-go/assistant/store"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	adkstore "github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// RuntimeOption Runtime 配置选项
type RuntimeOption func(*Runtime)

// WithStore 设置存储
func WithStore(s store.Store) RuntimeOption {
	return func(r *Runtime) {
		r.store = s
	}
}

// WithMaxRounds 设置保留的最大轮数，轮数按用户消息计数。
// 设为 <=0 表示不裁剪，保留全部历史。
func WithMaxRounds(n int) RuntimeOption {
	return func(r *Runtime) {
		r.maxRounds = n
	}
}

// Runtime 服务层核心，管理完整的对话生命周期
type Runtime struct {
	runner          *adk.Runner
	store           store.Store
	checkpointStore adk.CheckPointStore
	maxRounds       int // 保留最近 N 轮对话（按 user 消息计），<=0 表示不裁剪
}

// New 创建 Runtime
func New(agent adk.Agent, opts ...RuntimeOption) (*Runtime, error) {
	r := &Runtime{
		checkpointStore: adkstore.NewInMemoryStore(), // 默认内存 checkpoint
		maxRounds:       25,                          // 默认保留 25 轮
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
func (r *Runtime) Generate(ctx context.Context, messages []*schema.Message, checkpointID string) *adk.AsyncIterator[*adk.AgentEvent] {
	return r.runner.Run(ctx, messages, adk.WithCheckPointID(checkpointID))
}

// Events 执行对话，返回事件 channel（便利 API）
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
func (r *Runtime) Run(ctx context.Context, sess *session.Session, query string) error {
	// 1. 存储用户消息
	if err := r.store.Append(ctx, sess.ID, schema.UserMessage(query)); err != nil {
		return fmt.Errorf("append user message: %w", err)
	}

	// 2. 获取完整消息列表（含当前用户消息）
	history, err := r.store.Get(ctx, sess.ID)
	if err != nil {
		return fmt.Errorf("get history: %w", err)
	}

	// 2b. 裁剪历史到最近 N 轮
	history = trimRounds(history, r.maxRounds)

	// 3. 运行 agent
	_ = sess.Emit(session.Event{Type: session.TypeMessage, Content: "🤖: "})
	iter := r.Generate(ctx, history, sess.ID)

	// 4. 处理事件流
	messages, interruptInfo, err := r.drain(iter, sess)
	if err != nil {
		return fmt.Errorf("process events: %w", err)
	}

	// 5. 存储输出消息
	if len(messages) > 0 {
		if err = r.store.Append(ctx, sess.ID, messages...); err != nil {
			return fmt.Errorf("append messages: %w", err)
		}
	}

	// 6. 处理中断（审批）
	if interruptInfo != nil {
		return r.handleInterrupt(ctx, sess, interruptInfo)
	}

	// 7. 发送完成信号
	_ = sess.Emit(session.Event{Type: session.TypeMessage, Content: "\n"})

	return nil
}

// Resume 恢复中断的对话
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

// handleInterrupt 处理中断（审批）
func (r *Runtime) handleInterrupt(ctx context.Context, sess *session.Session, interruptInfo *adk.InterruptInfo) error {
	for _, ic := range interruptInfo.InterruptContexts {
		if !ic.IsRootCause {
			continue
		}

		// 发送审批请求
		approvalID := ic.ID
		info, ok := ic.Info.(*session.ApprovalInfo)
		if !ok {
			return fmt.Errorf("unexpected interrupt info type: %T", ic.Info)
		}

		_ = sess.Emit(session.Event{
			Type:    session.TypeApproval,
			Content: info.String(),
			Meta:    map[string]any{"approval_id": approvalID, "tool_name": info.ToolName},
		})

		// 等待审批结果
		input, err := sess.WaitInput(ctx)
		if err != nil {
			return fmt.Errorf("wait approval input: %w", err)
		}

		if input.Type != session.InputApproval {
			return fmt.Errorf("unexpected input type: %s", input.Type)
		}

		result, ok := input.Data.(*session.ApprovalResult)
		if !ok {
			return fmt.Errorf("unexpected approval result type: %T", input.Data)
		}

		// 校验 ApprovalID
		if result.ApprovalID != "" && result.ApprovalID != approvalID {
			return fmt.Errorf("approval ID mismatch: expected %s, got %s", approvalID, result.ApprovalID)
		}

		// 发送审批结果通知
		if result.Approved {
			_ = sess.Emit(session.Event{Type: session.TypeApprovalRes, Content: "✔️ Approved, executing...\n"})
		} else {
			_ = sess.Emit(session.Event{Type: session.TypeApprovalRes, Content: "✖️ Rejected\n"})
		}

		// 恢复执行
		messages, newInterrupt, err := r.Resume(ctx, sess, sess.ID, map[string]any{
			approvalID: result,
		})
		if err != nil {
			return fmt.Errorf("resume after approval: %w", err)
		}

		// 存储恢复后的消息
		if len(messages) > 0 {
			if err = r.store.Append(ctx, sess.ID, messages...); err != nil {
				return fmt.Errorf("append resumed messages: %w", err)
			}
		}

		// 递归处理后续中断
		if newInterrupt != nil {
			return r.handleInterrupt(ctx, sess, newInterrupt)
		}
	}
	return nil
}

// drain 消费事件迭代器并处理
func (r *Runtime) drain(iter *adk.AsyncIterator[*adk.AgentEvent], sess *session.Session) ([]*schema.Message, *adk.InterruptInfo, error) {
	messages := make([]*schema.Message, 0, 20)

	for {
		event, ok := iter.Next()
		if !ok {
			return messages, nil, nil
		}

		msg, interrupt, err := r.handleEvent(event, sess)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			messages = append(messages, msg)
		}
		if interrupt != nil {
			return messages, interrupt, nil
		}
	}
}

// handleEvent 处理单个事件
func (r *Runtime) handleEvent(event *adk.AgentEvent, sess *session.Session) (*schema.Message, *adk.InterruptInfo, error) {
	// 1. 错误
	if event.Err != nil {
		_ = sess.Emit(session.Event{Type: session.TypeMessage, Content: fmt.Sprintf("⚠️ %s\n", event.Err)})
		if errors.Is(event.Err, adk.ErrExceedMaxIterations) {
			return nil, nil, nil
		}
		return nil, nil, event.Err
	}

	// 2. 动作
	if event.Action != nil {
		return nil, r.handleAction(event.Action, sess), nil
	}

	// 3. 消息
	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil, nil, nil
	}

	return r.handleMessage(event.Output.MessageOutput, sess)
}

// handleAction 处理动作事件
func (r *Runtime) handleAction(action *adk.AgentAction, sess *session.Session) *adk.InterruptInfo {
	if action.Interrupted != nil {
		return action.Interrupted
	}
	if action.TransferToAgent != nil {
		_ = sess.Emit(session.Event{
			Type:    session.TypeMessage,
			Content: fmt.Sprintf("➡️ transfer to %s\n", action.TransferToAgent.DestAgentName),
		})
		return nil
	}
	if action.Exit {
		_ = sess.Emit(session.Event{Type: session.TypeMessage, Content: "🏁 exit\n"})
	}
	return nil
}

// handleMessage 处理消息事件
func (r *Runtime) handleMessage(mv *adk.MessageVariant, sess *session.Session) (*schema.Message, *adk.InterruptInfo, error) {
	// Tool 结果
	if mv.Role == schema.Tool {
		result, err := mv.GetMessage()
		if err != nil {
			return nil, nil, err
		}
		_ = sess.Emit(session.Event{
			Type:    session.TypeToolResult,
			Content: fmt.Sprintf("✅ [tool result] -> %s\n%s\n", mv.ToolName, truncate(result.Content, 200)),
		})
		return result, nil, nil
	}

	// Assistant
	if mv.Role != schema.Assistant && mv.Role != "" {
		return nil, nil, nil
	}

	if mv.IsStreaming {
		msg, err := r.handleStreamingMessage(mv, sess)
		return msg, nil, err
	}
	msg := r.handleRegularMessage(mv, sess)
	return msg, nil, nil
}

// handleStreamingMessage 处理流式消息
func (r *Runtime) handleStreamingMessage(mv *adk.MessageVariant, sess *session.Session) (*schema.Message, error) {
	mv.MessageStream.SetAutomaticClose()

	var contentBuf strings.Builder
	var reasoningBuf strings.Builder
	var toolCallMsgs []*schema.Message

	for {
		chunk, err := mv.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if chunk == nil {
			continue
		}

		if chunk.Content != "" {
			contentBuf.WriteString(chunk.Content)
			_ = sess.Emit(session.Event{Type: session.TypeAssistant, Content: chunk.Content})
		}

		if chunk.ReasoningContent != "" {
			reasoningBuf.WriteString(chunk.ReasoningContent)
			_ = sess.Emit(session.Event{Type: session.TypeReasoning, Content: chunk.ReasoningContent})
		}

		if len(chunk.ToolCalls) > 0 {
			for _, tc := range chunk.ToolCalls {
				toolCallMsgs = append(toolCallMsgs, &schema.Message{
					Role:      mv.Role,
					ToolCalls: []schema.ToolCall{tc},
				})
			}
		}
	}

	merged, _ := schema.ConcatMessages(toolCallMsgs)
	for _, tc := range merged.ToolCalls {
		_ = sess.Emit(session.Event{
			Type:    session.TypeToolCall,
			Content: fmt.Sprintf("\n🔧 [tool call] -> %s\t%s\n", tc.Function.Name, tc.Function.Arguments),
		})
	}

	return &schema.Message{
		Role:             schema.Assistant,
		Content:          contentBuf.String(),
		ReasoningContent: reasoningBuf.String(), // 如果思考过程不需要回传，可以忽略（deepseek需要）
		ToolCalls:        merged.ToolCalls,
	}, nil
}

// handleRegularMessage 处理非流式消息
func (r *Runtime) handleRegularMessage(mv *adk.MessageVariant, sess *session.Session) *schema.Message {
	if mv.Message == nil {
		return nil
	}

	if mv.Message.ReasoningContent != "" {
		_ = sess.Emit(session.Event{Type: session.TypeReasoning, Content: mv.Message.ReasoningContent})
	}

	_ = sess.Emit(session.Event{Type: session.TypeAssistant, Content: mv.Message.Content})

	for _, tc := range mv.Message.ToolCalls {
		_ = sess.Emit(session.Event{
			Type:    session.TypeToolCall,
			Content: fmt.Sprintf("\n🔧 [tool call] -> %s\t%s\n", tc.Function.Name, tc.Function.Arguments),
		})
	}
	return mv.Message
}

// trimRounds 只保留最近 maxRounds 轮对话（从末尾向前数 user 消息）
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

// truncate 截断字符串
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
