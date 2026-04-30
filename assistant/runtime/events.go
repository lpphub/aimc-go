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

func (r *Runtime) handleEvent(event *adk.AgentEvent, sess *session.Session) (*schema.Message, *adk.InterruptInfo, error) {
	if event.Err != nil {
		//if errors.Is(event.Err, adk.ErrExceedMaxIterations) {
		//	// max iterations
		//	return nil, nil, nil
		//}
		return nil, nil, event.Err
	}

	if event.Action != nil {
		return nil, r.handleAction(event.Action, sess), nil
	}

	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil, nil, nil
	}

	return r.handleMessage(event.Output.MessageOutput, sess)
}

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
	var toolCallMessages []*schema.Message

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
				toolCallMessages = append(toolCallMessages, &schema.Message{
					Role:      mv.Role,
					ToolCalls: []schema.ToolCall{tc},
				})
			}
		}
	}

	merged, _ := schema.ConcatMessages(toolCallMessages)
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
			_ = sess.Emit(session.Event{Type: session.TypeMessage, Content: "✔️ Approved, executing...\n"})
		} else {
			_ = sess.Emit(session.Event{Type: session.TypeMessage, Content: "✖️ Rejected\n"})
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
