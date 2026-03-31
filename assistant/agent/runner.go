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

type Runner struct {
	runner   *adk.Runner
	pipeline *EventPipeline
}

// NewRunner 创建 Runner
func NewRunner(agent adk.Agent) *Runner {
	return &Runner{
		runner: adk.NewRunner(context.Background(), adk.RunnerConfig{
			Agent:           agent,
			EnableStreaming: true,
		}),
		pipeline: NewEventPipeline(
			&ErrorHandler{},
			&ActionHandler{},
		),
	}
}

// Query 执行查询（便捷方法，用于单个用户消息）
func (r *Runner) Query(ctx context.Context, query string) (string, error) {
	fmt.Printf("用户: %s\n", query)

	iter := r.runner.Query(ctx, query)

	content, err := r.processEventStream(ctx, iter)
	if err != nil {
		return "", err
	}

	return content, nil
}

func (r *Runner) processEventStream(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder

	ec := &EventContext{
		Ctx:    ctx,
		Writer: &sb,
	}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if err := r.pipeline.Execute(ec, event); err != nil {
			return "", err
		}
	}

	return sb.String(), nil
}

func (r *Runner) handleEvent(event *adk.AgentEvent, sb *strings.Builder) error {
	// 1. error
	if event.Err != nil {
		return event.Err
	}

	// 2. action
	if event.Action != nil {
		r.handleAction(event.Action)
		return nil
	}

	// 3. output
	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil
	}

	mv := event.Output.MessageOutput

	switch mv.Role {
	case schema.Tool:
		r.handleToolResult(mv)
		return nil

	case schema.Assistant, "":
		return r.handleMessage(mv, sb)

	default:
		return nil
	}
}

func (r *Runner) handleMessage(mv *adk.MessageVariant, sb *strings.Builder) error {
	if mv.IsStreaming {
		return r.handleStreamingMessage(mv, sb)
	}
	return r.handleNonStreamingMessage(mv, sb)
}

func (r *Runner) handleStreamingMessage(mv *adk.MessageVariant, sb *strings.Builder) error {
	mv.MessageStream.SetAutomaticClose()

	var toolCalls []schema.ToolCall

	for {
		chunk, err := mv.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		if chunk == nil {
			continue
		}

		if chunk.Content != "" {
			sb.WriteString(chunk.Content)
			r.print(chunk.Content)
		}

		if len(chunk.ToolCalls) > 0 {
			toolCalls = append(toolCalls, chunk.ToolCalls...)
		}
	}

	// tool calls 统一处理
	r.printToolCalls(toolCalls)

	r.print("")
	return nil
}

func (r *Runner) handleNonStreamingMessage(mv *adk.MessageVariant, sb *strings.Builder) error {
	if mv.Message == nil {
		return nil
	}

	content := mv.Message.Content
	sb.WriteString(content)

	r.print(content)
	r.printToolCalls(mv.Message.ToolCalls)

	return nil
}

func (r *Runner) handleToolResult(mv *adk.MessageVariant) {
	result := r.drainToolResult(mv)
	r.printToolResult(mv.ToolName, result)
}

func (r *Runner) printToolCalls(toolCalls []schema.ToolCall) {
	for _, tc := range toolCalls {
		if tc.Function.Name != "" && tc.Function.Arguments != "" {
			r.printToolCall(tc.Function.Name, tc.Function.Arguments)
		}
	}
}

func (r *Runner) drainToolResult(mv *adk.MessageVariant) string {
	if mv.IsStreaming {
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

// handleAction 处理 Action 事件
func (r *Runner) handleAction(action *adk.AgentAction) {
	if action.Interrupted != nil {
		fmt.Printf("  ⏸️ [中断] 等待用户输入...\n")
	}
	if action.TransferToAgent != nil {
		fmt.Printf("  ➡️ [转移] 切换到 Agent: %s\n", action.TransferToAgent.DestAgentName)
	}
	if action.Exit {
		fmt.Printf("  🏁 [退出] Agent 执行结束\n")
	}
}

func (r *Runner) print(content string) {
	fmt.Printf("%s \n", content)
}

func (r *Runner) printError(err error) {
	fmt.Printf("  ❌ [错误] %v\n", err)
}

// printToolCallRequest 打印工具调用请求
func (r *Runner) printToolCall(name, arguments string) {
	fmt.Printf("  🔧 [调用工具] %s(%s)\n", name, r.truncate(arguments, 100))
}

// printToolResult 打印工具执行结果
func (r *Runner) printToolResult(toolName, result string) {
	truncated := len(result) > 200
	display := r.truncate(result, 200)
	if truncated {
		display += "..."
	}
	fmt.Printf("  ✅ [工具结果] %s: %s\n", toolName, display)
}

// truncate 截断字符串
func (r *Runner) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
