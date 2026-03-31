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
	Ctx    context.Context
	Writer *strings.Builder
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
		fmt.Printf("❌ %v\n", event.Err)
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
		fmt.Println("⏸️ interrupted")
	}

	if event.Action.TransferToAgent != nil {
		fmt.Printf("➡️ transfer to %s\n", event.Action.TransferToAgent.DestAgentName)
	}

	if event.Action.Exit {
		fmt.Println("🏁 exit")
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
	fmt.Printf("🔧 tool result [%s]: %s\n", mv.ToolName, truncate(result, 200))

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

	var content string
	var err error

	switch {
	case mv.IsStreaming:
		content, err = h.handleStreaming(ctx, mv)
	default:
		content, err = h.handleNonStreaming(ctx, mv)
	}

	if err != nil {
		return true, err
	}

	return true, nil
}

func (h *MessageHandler) handleStreaming(ec *EventContext, mv *adk.MessageVariant) (string, error) {
	mv.MessageStream.SetAutomaticClose()

	var sb strings.Builder
	var toolCalls []schema.ToolCall

	for {
		frame, err := mv.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if frame == nil {
			continue
		}

		if frame.Content != "" {
			sb.WriteString(frame.Content)
			ec.Writer.WriteString(frame.Content)
			fmt.Print(frame.Content)
		}

		if len(frame.ToolCalls) > 0 {
			toolCalls = append(toolCalls, frame.ToolCalls...)
		}
	}

	printToolCalls(toolCalls)
	fmt.Println()

	return sb.String(), nil
}

func (h *MessageHandler) handleNonStreaming(ec *EventContext, mv *adk.MessageVariant) (string, error) {
	if mv.Message == nil {
		return "", nil
	}

	content := mv.Message.Content
	ec.Writer.WriteString(content)

	fmt.Println(content)
	printToolCalls(mv.Message.ToolCalls)

	return content, nil
}
