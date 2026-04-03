package runtime

import (
	"aimc-go/assistant/channel"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// handleAgentEvent 处理单个 agent 事件，返回新产生的消息
func (r *Runtime) handleAgentEvent(ch *channel.Channel, event *adk.AgentEvent) (
	*schema.Message, *adk.InterruptInfo, error,
) {
	// 1. error
	if event.Err != nil {
		ch.Emit(channel.Chunk{Type: channel.TypeMessage, Content: fmt.Sprintf("⚠️ %s\n", event.Err)})
		// 不中断，当作正常结束
		if errors.Is(event.Err, adk.ErrExceedMaxIterations) {
			return nil, nil, nil
		}
		return nil, nil, event.Err
	}

	// 2. action
	if event.Action != nil {
		return nil, r.handleAction(ch, event.Action), nil
	}

	// 3. message
	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil, nil, nil
	}

	mv := event.Output.MessageOutput

	// tool result
	if mv.Role == schema.Tool {
		result, err := mv.GetMessage()
		if err != nil {
			return nil, nil, fmt.Errorf("get tool_result error: %w", err)
		}

		ch.Emit(channel.Chunk{
			Type:    channel.TypeToolResult,
			Content: fmt.Sprintf("✅ [tool result] -> %s: %s\n", mv.ToolName, r.truncate(result.Content, 200)),
		})
		return result, nil, nil
	}

	// 只处理 assistant
	if mv.Role != schema.Assistant && mv.Role != "" {
		return nil, nil, nil
	}

	if mv.IsStreaming {
		msg, err := r.handleStreaming(ch, mv)
		return msg, nil, err
	}
	return r.handleNonStreaming(ch, mv), nil, nil
}

// handleAction 处理 interrupt/transfer/exit
func (r *Runtime) handleAction(ch *channel.Channel, action *adk.AgentAction) *adk.InterruptInfo {
	if action.Interrupted != nil {
		ch.Emit(channel.Chunk{Type: channel.TypeMessage, Content: "⏸️ interrupted \n"})
		return action.Interrupted
	}

	if action.TransferToAgent != nil {
		ch.Emit(channel.Chunk{
			Type:    channel.TypeMessage,
			Content: fmt.Sprintf("➡️ transfer to %s\n", action.TransferToAgent.DestAgentName),
		})
		return nil
	}

	if action.Exit {
		ch.Emit(channel.Chunk{Type: channel.TypeMessage, Content: "🏁 exit\n"})
	}

	return nil
}

// handleStreaming 处理流式消息，返回完整消息
func (r *Runtime) handleStreaming(ch *channel.Channel, mv *adk.MessageVariant) (*schema.Message, error) {
	mv.MessageStream.SetAutomaticClose()

	var contentBuf strings.Builder
	var accumulatedToolCalls []schema.ToolCall

	for {
		frame, err := mv.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if frame == nil {
			continue
		}

		if frame.Content != "" {
			contentBuf.WriteString(frame.Content)
			ch.Emit(channel.Chunk{Type: channel.TypeAssistant, Content: frame.Content})
		}

		if len(frame.ToolCalls) > 0 {
			accumulatedToolCalls = append(accumulatedToolCalls, frame.ToolCalls...)
		}
	}

	// 换行
	ch.Emit(channel.Chunk{Type: channel.TypeMessage, Content: "\n"})

	// tool call 输出（展示）
	for _, tc := range accumulatedToolCalls {
		ch.Emit(channel.Chunk{
			Type:    channel.TypeToolCall,
			Content: fmt.Sprintf("🔧 [tool call] -> %s: %s\n", tc.Function.Name, r.truncate(tc.Function.Arguments, 200)),
		})
	}

	// 返回完整的 assistant 消息（content + ToolCalls）
	return &schema.Message{
		Role:      schema.Assistant,
		Content:   contentBuf.String(),
		ToolCalls: accumulatedToolCalls,
	}, nil
}

// handleNonStreaming 处理非流式消息，返回消息
func (r *Runtime) handleNonStreaming(ch *channel.Channel, mv *adk.MessageVariant) *schema.Message {
	if mv.Message == nil {
		return nil
	}

	// 输出展示
	ch.Emit(channel.Chunk{Type: channel.TypeAssistant, Content: mv.Message.Content})

	for _, tc := range mv.Message.ToolCalls {
		ch.Emit(channel.Chunk{
			Type:    channel.TypeToolCall,
			Content: fmt.Sprintf("\n🔧 [tool call] -> %s: %s\n", tc.Function.Name, r.truncate(tc.Function.Arguments, 200)),
		})
	}

	return mv.Message
}

// truncate 截断字符串，按 rune 截断避免破坏多字节字符
func (r *Runtime) truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen]) + "..."
}

// processEvents 迭代处理事件流，本地收集消息
func (r *Runtime) processEvents(ctx context.Context, ch *channel.Channel, iter *adk.AsyncIterator[*adk.AgentEvent]) (
	[]*schema.Message, *adk.InterruptInfo, error,
) {
	ch.Emit(channel.Chunk{Type: channel.TypeMessage, Content: "🤖: "})

	messages := make([]*schema.Message, 0, 20)

	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
			event, ok := iter.Next()
			if !ok {
				return messages, nil, nil // 迭代结束，返回收集的消息
			}

			msg, interruptInfo, err := r.handleAgentEvent(ch, event)
			if err != nil {
				return nil, nil, err
			}
			if msg != nil {
				messages = append(messages, msg)
			}
			if interruptInfo != nil {
				return messages, interruptInfo, nil
			}
		}
	}
}