package runtime

import (
	"aimc-go/assistant/session"
	"aimc-go/assistant/types"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func (r *Runtime) handleMessage(mv *adk.MessageVariant, sess *session.Session) (*schema.Message, *adk.InterruptInfo, error) {
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
		ReasoningContent: reasoningBuf.String(),
		ToolCalls:        merged.ToolCalls,
	}, nil
}

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

func (r *Runtime) handleInterrupt(ctx context.Context, sess *session.Session, interruptInfo *adk.InterruptInfo) error {
	for _, ic := range interruptInfo.InterruptContexts {
		if !ic.IsRootCause {
			continue
		}

		approvalID := ic.ID
		info, ok := ic.Info.(*types.ApprovalInfo)
		if !ok {
			return fmt.Errorf("unexpected interrupt info type: %T", ic.Info)
		}

		_ = sess.Emit(session.Event{
			Type:    session.TypeApproval,
			Content: info.String(),
			Meta:    map[string]any{"approval_id": approvalID, "tool_name": info.ToolName},
		})

		input, err := sess.WaitInput(ctx)
		if err != nil {
			return fmt.Errorf("wait approval input: %w", err)
		}

		if input.Type != session.InputApproval {
			return fmt.Errorf("unexpected input type: %s", input.Type)
		}

		result, ok := input.Data.(*types.ApprovalResult)
		if !ok {
			return fmt.Errorf("unexpected approval result type: %T", input.Data)
		}

		if result.ApprovalID != "" && result.ApprovalID != approvalID {
			return fmt.Errorf("approval ID mismatch: expected %s, got %s", approvalID, result.ApprovalID)
		}

		if result.Approved {
			_ = sess.Emit(session.Event{Type: session.TypeApprovalRes, Content: "✔️ Approved, executing...\n"})
		} else {
			_ = sess.Emit(session.Event{Type: session.TypeApprovalRes, Content: "✖️ Rejected\n"})
		}

		messages, newInterrupt, err := r.Resume(ctx, sess, sess.ID, map[string]any{
			approvalID: result,
		})
		if err != nil {
			return fmt.Errorf("resume after approval: %w", err)
		}

		if len(messages) > 0 {
			if err = r.store.Append(ctx, sess.ID, messages...); err != nil {
				return fmt.Errorf("append resumed messages: %w", err)
			}
		}

		if newInterrupt != nil {
			return r.handleInterrupt(ctx, sess, newInterrupt)
		}
	}
	return nil
}