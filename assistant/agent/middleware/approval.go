package middleware

import (
	"aimc-go/assistant/session"
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ApprovalMiddleware 审批中间件
type ApprovalMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	toolsToApprove map[string]bool // 需要审批的 Tool 名称
}

func NewApprovalMiddleware(toolsToApprove ...string) *ApprovalMiddleware {
	m := &ApprovalMiddleware{
		toolsToApprove: make(map[string]bool),
	}
	for _, t := range toolsToApprove {
		m.toolsToApprove[t] = true
	}
	return m
}

// checkApproval 检查审批状态。返回 resumeData 和审批结果（nil 表示需要触发/重新触发中断）
func (m *ApprovalMiddleware) checkApproval(ctx context.Context, args string) (storedArgs string, result *session.ApprovalResult) {
	wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
	if !wasInterrupted {
		return args, nil
	}
	_, _, result = tool.GetResumeContext[*session.ApprovalResult](ctx)
	return storedArgs, result
}

// WrapInvokableToolCall 拦截同步 Tool 调用
func (m *ApprovalMiddleware) WrapInvokableToolCall(_ context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	if !m.toolsToApprove[tCtx.Name] {
		return endpoint, nil
	}

	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		storedArgs, result := m.checkApproval(ctx, args)

		if result == nil {
			return "", tool.StatefulInterrupt(ctx, &session.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: storedArgs,
			}, storedArgs)
		}

		if result.Approved {
			return endpoint(ctx, storedArgs, opts...)
		}

		if result.DisapproveReason != nil {
			return fmt.Sprintf("tool '%s' disapproved: %s", tCtx.Name, *result.DisapproveReason), nil
		}
		return fmt.Sprintf("tool '%s' disapproved", tCtx.Name), nil
	}, nil
}

func (m *ApprovalMiddleware) WrapStreamableToolCall(_ context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	if !m.toolsToApprove[tCtx.Name] {
		return endpoint, nil
	}

	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		storedArgs, result := m.checkApproval(ctx, args)

		if result == nil {
			return nil, tool.StatefulInterrupt(ctx, &session.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: storedArgs,
			}, storedArgs)
		}

		if result.Approved {
			return endpoint(ctx, storedArgs, opts...)
		}

		if result.DisapproveReason != nil {
			return singleChunkReader(fmt.Sprintf("tool '%s' disapproved: %s", tCtx.Name, *result.DisapproveReason)), nil
		}
		return singleChunkReader(fmt.Sprintf("tool '%s' disapproved", tCtx.Name)), nil
	}, nil
}
