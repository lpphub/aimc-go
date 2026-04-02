package runtime

import (
	"aimc-go/assistant/approval"
	"aimc-go/assistant/sink"
	"aimc-go/assistant/store"
	"context"
	"fmt"

	adkstore "github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// Runtime 服务层核心，管理完整的对话生命周期
type Runtime struct {
	agent           adk.Agent
	store           store.Store
	checkpointStore adk.CheckPointStore
}

// RuntimeOption Runtime 配置选项
type RuntimeOption func(*Runtime)

// WithStore 设置存储
func WithStore(s store.Store) RuntimeOption {
	return func(r *Runtime) {
		r.store = s
	}
}

// WithCheckpointStore 设置 checkpoint 存储
func WithCheckpointStore(cs adk.CheckPointStore) RuntimeOption {
	return func(r *Runtime) {
		r.checkpointStore = cs
	}
}

// NewRuntime 创建 Runtime
func NewRuntime(agent adk.Agent, opts ...RuntimeOption) (*Runtime, error) {
	r := &Runtime{
		agent:           agent,
		checkpointStore: adkstore.NewInMemoryStore(), // 默认内存 checkpoint
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.store == nil {
		return nil, fmt.Errorf("store is required, use WithStore() to set")
	}

	return r, nil
}

// Run 执行一轮对话
func (r *Runtime) Run(ctx context.Context, session *Session, query string) error {
	// 1. 加载/创建会话历史
	sessHistory, err := r.store.GetOrCreate(ctx, session.ID)
	if err != nil {
		return fmt.Errorf("get or create session: %w", err)
	}

	err = r.store.Append(ctx, session.ID, schema.UserMessage(query))
	if err != nil {
		return fmt.Errorf("append user message: %w", err)
	}

	// 2. 运行 agent，获取事件流
	innerRunner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           r.agent,
		EnableStreaming: true,
		CheckPointStore: r.checkpointStore,
	})

	iter := innerRunner.Run(ctx, sessHistory.Messages, adk.WithCheckPointID(session.ID))

	// 3. 处理事件流
	messages, interruptInfo, err := r.processEvents(ctx, session, iter)
	if err != nil {
		return err
	}

	// 4. 存储输出消息
	if len(messages) > 0 {
		err = r.store.Append(ctx, session.ID, messages...)
		if err != nil {
			return fmt.Errorf("append messages: %w", err)
		}
	}

	// 5. 处理中断（审批）
	if interruptInfo != nil {
		return r.handleInterrupt(ctx, session, interruptInfo)
	}

	// 6. 发送完成信号
	session.Emit(sink.Chunk{Type: sink.TypeDone})

	return nil
}

// handleInterrupt 处理中断（审批）
func (r *Runtime) handleInterrupt(ctx context.Context, session *Session, interruptInfo *adk.InterruptInfo) error {
	for _, ic := range interruptInfo.InterruptContexts {
		if !ic.IsRootCause {
			continue
		}

		// 发送审批请求
		approvalID := ic.ID
		info, ok := ic.Info.(*approval.ApprovalInfo)
		if !ok {
			return fmt.Errorf("unexpected interrupt info type: %T", ic.Info)
		}

		session.Emit(sink.Chunk{
			Type:    sink.TypeApproval,
			Content: info.String(),
			Meta:    map[string]any{"approval_id": approvalID, "tool_name": info.ToolName},
		})

		// 等待审批结果
		input, err := session.WaitInput(ctx)
		if err != nil {
			return fmt.Errorf("wait approval input: %w", err)
		}

		if input.Type != InputApproval {
			return fmt.Errorf("unexpected input type: %s", input.Type)
		}

		result, ok := input.Data.(*approval.ApprovalResult)
		if !ok {
			return fmt.Errorf("unexpected approval result type: %T", input.Data)
		}

		// 发送审批结果通知
		if result.Approved {
			session.Emit(sink.Chunk{Type: sink.TypeApprovalRes, Content: "✔️ Approved, executing...\n"})
		} else {
			session.Emit(sink.Chunk{Type: sink.TypeApprovalRes, Content: "✖️ Rejected\n"})
		}

		// 恢复执行
		messages, newInterrupt, err := r.Resume(ctx, session, session.ID, map[string]any{
			approvalID: result,
		})
		if err != nil {
			return fmt.Errorf("resume after approval: %w", err)
		}

		// 存储恢复后的消息
		if len(messages) > 0 {
			err = r.store.Append(ctx, session.ID, messages...)
			if err != nil {
				return fmt.Errorf("append resumed messages: %w", err)
			}
		}

		// 递归处理后续中断
		if newInterrupt != nil {
			return r.handleInterrupt(ctx, session, newInterrupt)
		}
	}

	// 发送完成信号
	session.Emit(sink.Chunk{Type: sink.TypeDone})

	return nil
}

// Resume 恢复中断的对话
func (r *Runtime) Resume(ctx context.Context, session *Session, checkpointID string, resumeData map[string]any) (
	[]*schema.Message, *adk.InterruptInfo, error,
) {
	innerRunner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           r.agent,
		EnableStreaming: true,
		CheckPointStore: r.checkpointStore,
	})

	events, err := innerRunner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{
		Targets: resumeData,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("resume with params: %w", err)
	}

	return r.processEvents(ctx, session, events)
}