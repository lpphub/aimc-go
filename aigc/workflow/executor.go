package workflow

import (
	"aimc-go/aigc/core"
	"context"
	"fmt"
)

type Executor struct {
	client *core.Client
	tools  map[string]core.ToolFunc
}

func NewExecutor(client *core.Client, tools map[string]core.ToolFunc) *Executor {
	if tools == nil {
		tools = make(map[string]core.ToolFunc)
	}
	return &Executor{client: client, tools: tools}
}

type Result struct {
	Output      any
	NodeOutputs map[string]any
}

func (e *Executor) Run(ctx context.Context, g *Graph, input map[string]any) (*Result, error) {
	nodeOutputs := make(map[string]any)

	currentID := g.Start
	for currentID != "" {
		node, ok := g.Nodes[currentID]
		if !ok {
			return nil, fmt.Errorf("node not found: %s", currentID)
		}

		output, err := e.executeNode(ctx, node, nodeOutputs, input)
		if err != nil {
			return nil, fmt.Errorf("node %s failed: %w", currentID, err)
		}

		nodeOutputs[currentID] = output
		currentID = e.nextNode(g, currentID)
	}

	return &Result{
		Output:      nodeOutputs[g.Start],
		NodeOutputs: nodeOutputs,
	}, nil
}

func (e *Executor) executeNode(ctx context.Context, node *Node, outputs map[string]any, input map[string]any) (any, error) {
	switch node.Type {
	case NodeTypeLLM:
		return e.executeLLM(ctx, node, outputs, input)
	case NodeTypeTool:
		return e.executeTool(ctx, node, outputs, input)
	default:
		return nil, fmt.Errorf("unsupported node type: %s", node.Type)
	}
}

func (e *Executor) executeLLM(ctx context.Context, node *Node, outputs map[string]any, input map[string]any) (any, error) {
	resp, err := e.client.Generate(ctx, &core.GenerateRequest{
		Task:   node.Config.Task,
		Prompt: node.Config.Prompt,
	})
	if err != nil {
		return nil, err
	}
	return resp.Text, nil
}

func (e *Executor) executeTool(ctx context.Context, node *Node, outputs map[string]any, input map[string]any) (any, error) {
	fn, ok := e.tools[node.Config.ToolName]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", node.Config.ToolName)
	}
	return fn(ctx, "")
}

func (e *Executor) nextNode(g *Graph, currentID string) string {
	for _, edge := range g.Edges {
		if edge.From == currentID {
			return edge.To
		}
	}
	return ""
}
