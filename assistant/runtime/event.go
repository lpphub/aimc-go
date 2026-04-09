package runtime

import (
	"aimc-go/assistant/channel"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// EventHandler 处理 adk.AgentEvent 并写入 channel.Channel（无状态）
type EventHandler struct{}

// HandleEvent 处理单个事件，返回：(产生的消息, 中断信息, 错误)
func (h *EventHandler) HandleEvent(event *adk.AgentEvent, ch *channel.Channel) (*schema.Message, *adk.InterruptInfo, error) {
	// 1. 错误
	if event.Err != nil {
		_ = ch.Write(channel.Chunk{Type: channel.TypeMessage, Content: fmt.Sprintf("⚠️ %s\n", event.Err)})
		if errors.Is(event.Err, adk.ErrExceedMaxIterations) {
			return nil, nil, nil
		}
		return nil, nil, event.Err
	}

	// 2. 动作
	if event.Action != nil {
		return nil, h.handleAction(event.Action, ch), nil
	}

	// 3. 消息
	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil, nil, nil
	}

	return h.handleMessage(event.Output.MessageOutput, ch)
}

func (h *EventHandler) handleAction(action *adk.AgentAction, ch *channel.Channel) *adk.InterruptInfo {
	if action.Interrupted != nil {
		return action.Interrupted
	}
	if action.TransferToAgent != nil {
		_ = ch.Write(channel.Chunk{
			Type:    channel.TypeMessage,
			Content: fmt.Sprintf("➡️ transfer to %s\n", action.TransferToAgent.DestAgentName),
		})
		return nil
	}
	if action.Exit {
		_ = ch.Write(channel.Chunk{Type: channel.TypeMessage, Content: "🏁 exit\n"})
	}
	return nil
}

func (h *EventHandler) handleMessage(mv *adk.MessageVariant, ch *channel.Channel) (*schema.Message, *adk.InterruptInfo, error) {
	// Tool 结果
	if mv.Role == schema.Tool {
		result, err := mv.GetMessage()
		if err != nil {
			return nil, nil, err
		}
		_ = ch.Write(channel.Chunk{
			Type:    channel.TypeToolResult,
			Content: fmt.Sprintf("✅ [tool result] -> %s\t%s\n", mv.ToolName, h.truncate(result.Content, 200)),
		})
		return result, nil, nil
	}

	// Assistant
	if mv.Role != schema.Assistant && mv.Role != "" {
		return nil, nil, nil
	}

	if mv.IsStreaming {
		msg, err := h.handleStreaming(mv, ch)
		return msg, nil, err
	}
	msg := h.handleRegular(mv, ch)
	return msg, nil, nil
}

func (h *EventHandler) handleStreaming(mv *adk.MessageVariant, ch *channel.Channel) (*schema.Message, error) {
	mv.MessageStream.SetAutomaticClose()

	var contentBuf strings.Builder
	var toolCalls []schema.ToolCall

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
			if err := ch.Write(channel.Chunk{Type: channel.TypeAssistant, Content: frame.Content}); err != nil {
				return nil, err
			}
		}
		if len(frame.ToolCalls) > 0 {
			toolCalls = append(toolCalls, frame.ToolCalls...)
		}
	}

	_ = ch.Write(channel.Chunk{Type: channel.TypeMessage, Content: "\n"})

	for _, tc := range toolCalls {
		_ = ch.Write(channel.Chunk{
			Type:    channel.TypeToolCall,
			Content: fmt.Sprintf("🔧 [tool call] -> %s\t%s\n", tc.Function.Name, tc.Function.Arguments),
		})
	}

	return &schema.Message{
		Role:      schema.Assistant,
		Content:   contentBuf.String(),
		ToolCalls: toolCalls,
	}, nil
}

func (h *EventHandler) handleRegular(mv *adk.MessageVariant, ch *channel.Channel) *schema.Message {
	if mv.Message == nil {
		return nil
	}
	_ = ch.Write(channel.Chunk{Type: channel.TypeAssistant, Content: mv.Message.Content})

	for _, tc := range mv.Message.ToolCalls {
		_ = ch.Write(channel.Chunk{
			Type:    channel.TypeToolCall,
			Content: fmt.Sprintf("\n🔧 [tool call] -> %s\t%s\n", tc.Function.Name, tc.Function.Arguments),
		})
	}
	return mv.Message
}

// Drain 消费事件迭代器并处理
func (h *EventHandler) Drain(iter *adk.AsyncIterator[*adk.AgentEvent], ch *channel.Channel) ([]*schema.Message, *adk.InterruptInfo, error) {
	messages := make([]*schema.Message, 0, 20)

	for {
		event, ok := iter.Next()
		if !ok {
			return messages, nil, nil
		}

		msg, interrupt, err := h.HandleEvent(event, ch)
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

func (h *EventHandler) truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
