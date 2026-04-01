package agent

import (
	"aimc-go/assistant/sink"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type EventContext struct {
	Ctx       context.Context
	Collector *strings.Builder // 收集assistant, 后续可扩展兼容工具调用
	Sink      sink.Sink        // 输出sink: std，sse
}

type EventHandler struct {
}

func (e *EventHandler) HandleEvent(ec *EventContext, event *adk.AgentEvent) (*adk.InterruptInfo, error) {
	// 1. error
	if event.Err != nil {
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
		result := e.drainToolResult(mv)

		ec.Sink.Output(sink.Event{
			Type:    "tool_result",
			Content: fmt.Sprintf("✅ tool result [%s]: %s\n", mv.ToolName, e.truncate(result, 200))},
		)
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
		ec.Sink.Output(sink.Event{Type: "log", Content: "⏸️ interrupted \n"})
		return action.Interrupted
	}

	if action.TransferToAgent != nil {
		ec.Sink.Output(sink.Event{
			Type:    "log",
			Content: fmt.Sprintf("➡️ transfer to %s\n", action.TransferToAgent.DestAgentName),
		})
		return nil
	}

	if action.Exit {
		ec.Sink.Output(sink.Event{Type: "log", Content: "🏁 exit\n"})
	}

	return nil
}

func (e *EventHandler) handleStreaming(ec *EventContext, mv *adk.MessageVariant) error {
	mv.MessageStream.SetAutomaticClose()

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
			// 收集 assistant 消息
			ec.Collector.WriteString(frame.Content)
			ec.Sink.Output(sink.Event{Type: "assistant", Content: frame.Content})
		}

		if len(frame.ToolCalls) > 0 {
			accumulatedToolCalls = append(accumulatedToolCalls, frame.ToolCalls...)
		}
	}

	// tool call（统一输出）
	for _, tc := range accumulatedToolCalls {
		ec.Sink.Output(sink.Event{
			Type:    "tool_call",
			Content: fmt.Sprintf("\n🔧 tool call [%s]: %s\n", tc.Function.Name, e.truncate(tc.Function.Arguments, 200)),
		})
	}

	// 换行
	ec.Sink.Output(sink.Event{Type: "message", Content: "\n"})

	return nil
}

func (e *EventHandler) handleNonStreaming(ec *EventContext, mv *adk.MessageVariant) error {
	if mv.Message == nil {
		return nil
	}

	content := mv.Message.Content

	ec.Collector.WriteString(content)
	ec.Sink.Output(sink.Event{Type: "assistant", Content: content})

	for _, tc := range mv.Message.ToolCalls {
		ec.Sink.Output(sink.Event{
			Type:    "tool_call",
			Content: fmt.Sprintf("🔧 tool call [%s]: %s\n", tc.Function.Name, e.truncate(tc.Function.Arguments, 200)),
		})
	}

	return nil
}

func (e *EventHandler) drainToolResult(mv *adk.MessageVariant) string {
	if mv.IsStreaming && mv.MessageStream != nil {
		var sb strings.Builder
		for {
			chunk, err := mv.MessageStream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				break
			}
			if chunk != nil && chunk.Content != "" {
				sb.WriteString(chunk.Content)
			}
		}
		return sb.String()
	}
	if mv.Message != nil {
		return mv.Message.Content
	}
	return ""
}

func (e *EventHandler) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
