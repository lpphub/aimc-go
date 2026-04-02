package agent

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

// EventContext 一轮对话的上下文
type EventContext struct {
	Ctx      context.Context
	messages []*schema.Message
	Sink     sink.Sink // 可选的输出通道
}

// NewEventContext 创建事件上下文
func NewEventContext(ctx context.Context, s sink.Sink) *EventContext {
	return &EventContext{
		Ctx:      ctx,
		messages: make([]*schema.Message, 0, 20), // 预分配，减少扩容
		Sink:     s,
	}
}

// Collect 收集完整消息
func (ec *EventContext) Collect(msg *schema.Message) {
	ec.messages = append(ec.messages, msg)
}

// Messages 返回收集的所有消息
func (ec *EventContext) Messages() []*schema.Message {
	return ec.messages
}

// Emit 发射输出片段
func (ec *EventContext) Emit(c sink.Chunk) {
	if ec.Sink != nil {
		ec.Sink.Emit(c)
	}
}

type EventHandler struct{}

func (e *EventHandler) HandleEvent(ec *EventContext, event *adk.AgentEvent) (*adk.InterruptInfo, error) {
	// 1. error
	if event.Err != nil {
		ec.Emit(sink.Chunk{Kind: sink.KindMessage, Content: fmt.Sprintf("⚠️ %s\n", event.Err)})
		// 不中断，当作正常结束
		if errors.Is(event.Err, adk.ErrExceedMaxIterations) {
			return nil, nil
		}
		return nil, event.Err
	}

	// 2. action
	if event.Action != nil {
		return e.handleAction(ec, event.Action), nil
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
		ec.Collect(result)

		ec.Emit(sink.Chunk{
			Kind:    sink.KindToolResult,
			Content: fmt.Sprintf("✅ [tool result] -> %s: %s\n", mv.ToolName, e.truncate(result.Content, 200)),
		})
		return nil, nil
	}

	// 只处理 assistant
	if mv.Role != schema.Assistant && mv.Role != "" {
		return nil, nil
	}

	if mv.IsStreaming {
		return nil, e.handleStreaming(ec, mv)
	}
	return nil, e.handleNonStreaming(ec, mv)
}

func (e *EventHandler) handleAction(ec *EventContext, action *adk.AgentAction) *adk.InterruptInfo {
	if action.Interrupted != nil {
		ec.Emit(sink.Chunk{Kind: sink.KindMessage, Content: "⏸️ interrupted \n"})
		return action.Interrupted
	}

	if action.TransferToAgent != nil {
		ec.Emit(sink.Chunk{
			Kind:    sink.KindMessage,
			Content: fmt.Sprintf("➡️ transfer to %s\n", action.TransferToAgent.DestAgentName),
		})
		return nil
	}

	if action.Exit {
		ec.Emit(sink.Chunk{Kind: sink.KindMessage, Content: "🏁 exit\n"})
	}

	return nil
}

func (e *EventHandler) handleStreaming(ec *EventContext, mv *adk.MessageVariant) error {
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
			ec.Emit(sink.Chunk{Kind: sink.KindAssistant, Content: frame.Content})
		}

		if len(frame.ToolCalls) > 0 {
			accumulatedToolCalls = append(accumulatedToolCalls, frame.ToolCalls...)
		}
	}

	// 换行
	ec.Emit(sink.Chunk{Kind: sink.KindMessage, Content: "\n"})

	// tool call 输出（展示）
	for _, tc := range accumulatedToolCalls {
		ec.Emit(sink.Chunk{
			Kind:    sink.KindToolCall,
			Content: fmt.Sprintf("🔧 [tool call] -> %s: %s\n", tc.Function.Name, e.truncate(tc.Function.Arguments, 200)),
		})
	}

	// 收集完整的 assistant 消息（content + ToolCalls）
	ec.Collect(&schema.Message{
		Role:      schema.Assistant,
		Content:   contentBuf.String(),
		ToolCalls: accumulatedToolCalls,
	})

	return nil
}

func (e *EventHandler) handleNonStreaming(ec *EventContext, mv *adk.MessageVariant) error {
	if mv.Message == nil {
		return nil
	}

	// 输出展示
	ec.Emit(sink.Chunk{Kind: sink.KindAssistant, Content: mv.Message.Content})

	for _, tc := range mv.Message.ToolCalls {
		ec.Emit(sink.Chunk{
			Kind:    sink.KindToolCall,
			Content: fmt.Sprintf("\n🔧 [tool call] -> %s: %s\n", tc.Function.Name, e.truncate(tc.Function.Arguments, 200)),
		})
	}

	// 收集完整的 assistant 消息
	ec.Collect(mv.Message)

	return nil
}

func (e *EventHandler) truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	// 按 rune 截断，避免破坏多字节字符
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return s
}
