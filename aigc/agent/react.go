package agent

import (
	"aimc-go/aigc/core"
	"context"
	"fmt"
)

type ReactAgent struct {
	config *AgentConfig
	client *core.Client
}

func (a *ReactAgent) ID() string {
	return a.config.Name
}

func (a *ReactAgent) Run(ctx context.Context, input string) (*AgentResult, error) {
	messages := []core.Message{
		{Role: "system", Content: a.config.SystemPrompt},
		{Role: "user", Content: input},
	}

	steps := []AgentStep{
		{Role: "user", Content: input},
	}

	for turn := 0; turn < a.config.MaxTurns; turn++ {
		resp, err := a.client.Generate(ctx, &core.GenerateRequest{
			Model:    a.config.Model,
			Messages: messages,
			Tools:    a.config.Tools,
		})

		if err != nil {
			return nil, fmt.Errorf("turn %d failed: %w", turn, err)
		}

		if len(resp.ToolCalls) == 0 {
			steps = append(steps, AgentStep{Role: "assistant", Content: resp.Text})
			return &AgentResult{
				FinalOutput: resp.Text,
				Steps:       steps,
			}, nil
		}

		messages = append(messages, core.Message{Role: "assistant", ToolCalls: resp.ToolCalls})
		steps = append(steps, AgentStep{Role: "assistant", ToolCall: &resp.ToolCalls[0]})

		for _, tc := range resp.ToolCalls {
			fn, ok := a.config.ToolFuncs[tc.Name]
			if !ok {
				return nil, fmt.Errorf("tool not found: %s", tc.Name)
			}

			result, err := fn(ctx, tc.Arguments)
			if err != nil {
				return nil, fmt.Errorf("tool %s failed: %w", tc.Name, err)
			}

			messages = append(messages, core.Message{Role: "tool", Content: result})
			steps = append(steps, AgentStep{Role: "tool", Content: result})
		}
	}

	return nil, fmt.Errorf("max turns (%d) exceeded", a.config.MaxTurns)
}

func (a *ReactAgent) RunStream(ctx context.Context, input string) (<-chan AgentEvent, error) {
	ch := make(chan AgentEvent)

	go func() {
		defer close(ch)

		result, err := a.Run(ctx, input)
		if err != nil {
			ch <- AgentEvent{Type: "error", Content: err.Error(), Done: true}
			return
		}

		ch <- AgentEvent{Type: "content", Content: result.FinalOutput, Done: true}
	}()

	return ch, nil
}
