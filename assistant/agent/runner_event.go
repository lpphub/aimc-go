package agent

import (
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
	Collector *strings.Builder // for assistant
	Writer    *MsgWriter       // assistant output
}

type EventHandler interface {
	Handle(ctx *EventContext, event *adk.AgentEvent) (bool, error)
}

type EventPipeline struct {
	handlers []EventHandler
}

func NewEventPipeline(handlers ...EventHandler) *EventPipeline {
	return &EventPipeline{
		handlers: handlers,
	}
}

func (p *EventPipeline) Execute(ctx *EventContext, event *adk.AgentEvent) error {
	for _, h := range p.handlers {
		handled, err := h.Handle(ctx, event)
		if err != nil {
			return err
		}
		if handled {
			return nil
		}
	}
	return nil
}

type ErrorHandler struct{}

func (h *ErrorHandler) Handle(ctx *EventContext, event *adk.AgentEvent) (bool, error) {
	if event.Err != nil {
		ctx.Writer.Output("❌ %v\n", event.Err)
		return true, event.Err
	}
	return false, nil
}

type ActionHandler struct{}

func (h *ActionHandler) Handle(ctx *EventContext, event *adk.AgentEvent) (bool, error) {
	if event.Action == nil {
		return false, nil
	}

	if event.Action.Interrupted != nil {
		ctx.Writer.Output("%s \n", "⏸️ interrupted")
	}

	if event.Action.TransferToAgent != nil {
		ctx.Writer.Output("➡️ transfer to %s\n", event.Action.TransferToAgent.DestAgentName)
	}

	if event.Action.Exit {
		ctx.Writer.Output("%s \n", "🏁 exit")
	}

	return true, nil
}

type ToolHandler struct{}

func (h *ToolHandler) Handle(ctx *EventContext, event *adk.AgentEvent) (bool, error) {
	if event.Output == nil || event.Output.MessageOutput == nil {
		return false, nil
	}

	mv := event.Output.MessageOutput

	if mv.Role != schema.Tool {
		return false, nil
	}

	result := drainToolResult(mv)

	ctx.Writer.Output("🔧 tool result [%s]: %s\n", mv.ToolName, truncate(result, 200))

	return true, nil
}

type MessageHandler struct{}

func (h *MessageHandler) Handle(ctx *EventContext, event *adk.AgentEvent) (bool, error) {
	if event.Output == nil || event.Output.MessageOutput == nil {
		return false, nil
	}

	mv := event.Output.MessageOutput

	if mv.Role != schema.Assistant && mv.Role != "" {
		return false, nil
	}

	var err error

	switch {
	case mv.IsStreaming:
		err = h.handleStreaming(ctx, mv)
	default:
		err = h.handleNonStreaming(ctx, mv)
	}

	if err != nil {
		return true, err
	}

	return true, nil
}

func (h *MessageHandler) handleStreaming(ec *EventContext, mv *adk.MessageVariant) error {
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
			ec.Collector.WriteString(frame.Content)
			ec.Writer.Output("%s", frame.Content)
		}

		if len(frame.ToolCalls) > 0 {
			accumulatedToolCalls = append(accumulatedToolCalls, frame.ToolCalls...)
		}
	}

	if len(accumulatedToolCalls) > 0 {
		for _, tc := range accumulatedToolCalls {
			ec.Writer.Output("🔧 tool call [%s]: %s\n", tc.Function.Name, truncate(tc.Function.Arguments, 200))
		}
	}

	ec.Writer.Output("\n")

	return nil
}

func (h *MessageHandler) handleNonStreaming(ec *EventContext, mv *adk.MessageVariant) error {
	if mv.Message == nil {
		return nil
	}

	content := mv.Message.Content
	ec.Collector.WriteString(content)

	ec.Writer.Output("%s", content)

	for _, tc := range mv.Message.ToolCalls {
		ec.Writer.Output("🔧 tool call [%s]: %s\n", tc.Function.Name, truncate(tc.Function.Arguments, 200))
	}

	return nil
}

func drainToolResult(mo *adk.MessageVariant) string {
	if mo.IsStreaming && mo.MessageStream != nil {
		var sb strings.Builder
		for {
			chunk, err := mo.MessageStream.Recv()
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
	if mo.Message != nil {
		return mo.Message.Content
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

type MsgWriter struct{}

func (po *MsgWriter) Output(format string, args ...any) {
	fmt.Printf(format, args...)
}
