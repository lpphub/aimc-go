package workflow

import (
	"aimc-go/aigc/core"
	"context"
	"encoding/json"
)

func (g *Graph) AsTool(name, desc string) core.ToolDefinition {
	return core.ToolDefinition{
		Name:        name,
		Description: desc,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type":        "string",
					"description": "Input for the workflow",
				},
			},
		},
	}
}

func (e *Executor) ToolFunc(g *Graph) core.ToolFunc {
	return func(ctx context.Context, args string) (string, error) {
		var input map[string]any
		if err := json.Unmarshal([]byte(args), &input); err != nil {
			input = map[string]any{"input": args}
		}

		result, err := e.Run(ctx, g, input)
		if err != nil {
			return "", err
		}

		output, _ := json.Marshal(result.Output)
		return string(output), nil
	}
}
