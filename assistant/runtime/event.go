package runtime

import (
	"aimc-go/assistant/sink"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// Runtime 处理 eino AgentEvent 并转换为 Chunk 输出
type Runtime struct{}

// handleAgentEvent 处理单个 agent 事件
func (r *Runtime) handleAgentEvent(session *Session, event *adk.AgentEvent) (*adk.InterruptInfo, error) {
	// 1. error
	if event.Err != nil {
		session.Emit(sink.Chunk{Type: sink.TypeMessage, Content: fmt.Sprintf("⚠️ %s\n", event.Err)})
		// 不中断，当作正常结束
		if errors.Is(event.Err, adk.ErrExceedMaxIterations) {
			return nil, nil
		}
		return nil, event.Err
	}

	// 2. action
	if event.Action != nil {
		return r.handleAction(session, event.Action), nil
	}

	// 3. message
	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil, nil
	}

	mv := event.Output.MessageOutput

	// tool result
	if mv.Role == schema.Tool {
		result, err := mv.GetMessage()
		if err != nil {
			return nil, fmt.Errorf("get tool_result error: %w", err)
		}

		// 收集完整的 tool result 消息
		session.Collect(result)

		session.Emit(sink.Chunk{
			Type:    sink.TypeToolResult,
			Content: fmt.Sprintf("✅ [tool result] -> %s: %s\n", mv.ToolName, r.truncate(result.Content, 200)),
		})
		return nil, nil
	}

	// 只处理 assistant
	if mv.Role != schema.Assistant && mv.Role != "" {
		return nil, nil
	}

	if mv.IsStreaming {
		return nil, r.handleStreaming(session, mv)
	}
	return nil, r.handleNonStreaming(session, mv)
}

// handleAction 处理 interrupt/transfer/exit
func (r *Runtime) handleAction(session *Session, action *adk.AgentAction) *adk.InterruptInfo {
	if action.Interrupted != nil {
		session.Emit(sink.Chunk{Type: sink.TypeMessage, Content: "⏸️ interrupted \n"})
		return action.Interrupted
	}

	if action.TransferToAgent != nil {
		session.Emit(sink.Chunk{
			Type:    sink.TypeMessage,
			Content: fmt.Sprintf("➡️ transfer to %s\n", action.TransferToAgent.DestAgentName),
		})
		return nil
	}

	if action.Exit {
		session.Emit(sink.Chunk{Type: sink.TypeMessage, Content: "🏁 exit\n"})
	}

	return nil
}

// handleStreaming 处理流式消息
func (r *Runtime) handleStreaming(session *Session, mv *adk.MessageVariant) error {
	mv.MessageStream.SetAutomaticClose()

	var contentBuf strings.Builder
	var accumulatedToolCalls []schema.ToolCall

	for {
		frame, err := mv.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if frame == nil {
			continue
		}

		if frame.Content != "" {
			contentBuf.WriteString(frame.Content)
			session.Emit(sink.Chunk{Type: sink.TypeAssistant, Content: frame.Content})
		}

		if len(frame.ToolCalls) > 0 {
			accumulatedToolCalls = append(accumulatedToolCalls, frame.ToolCalls...)
		}
	}

	// 换行
	session.Emit(sink.Chunk{Type: sink.TypeMessage, Content: "\n"})

	// tool call 输出（展示）
	for _, tc := range accumulatedToolCalls {
		session.Emit(sink.Chunk{
			Type:    sink.TypeToolCall,
			Content: fmt.Sprintf("🔧 [tool call] -> %s: %s\n", tc.Function.Name, r.truncate(tc.Function.Arguments, 200)),
		})
	}

	// 收集完整的 assistant 消息（content + ToolCalls）
	session.Collect(&schema.Message{
		Role:      schema.Assistant,
		Content:   contentBuf.String(),
		ToolCalls: accumulatedToolCalls,
	})

	return nil
}

// handleNonStreaming 处理非流式消息
func (r *Runtime) handleNonStreaming(session *Session, mv *adk.MessageVariant) error {
	if mv.Message == nil {
		return nil
	}

	// 输出展示
	session.Emit(sink.Chunk{Type: sink.TypeAssistant, Content: mv.Message.Content})

	for _, tc := range mv.Message.ToolCalls {
		session.Emit(sink.Chunk{
			Type:    sink.TypeToolCall,
			Content: fmt.Sprintf("\n🔧 [tool call] -> %s: %s\n", tc.Function.Name, r.truncate(tc.Function.Arguments, 200)),
		})
	}

	// 收集完整的 assistant 消息
	session.Collect(mv.Message)

	return nil
}

// truncate 截断字符串，按 rune 截断避免破坏多字节字符
func (r *Runtime) truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen]) + "..."
}

// processEvents 迭代处理事件流
func (r *Runtime) processEvents(ctx context.Context, session *Session, iter *adk.AsyncIterator[*adk.AgentEvent]) (*adk.InterruptInfo, error) {
	session.Emit(sink.Chunk{Type: sink.TypeMessage, Content: "🤖: "})

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			event, ok := iter.Next()
			if !ok {
				break
			}

			interruptInfo, err := r.handleAgentEvent(session, event)
			if err != nil {
				return nil, err
			}
			if interruptInfo != nil {
				return interruptInfo, nil
			}
		}
	}
}