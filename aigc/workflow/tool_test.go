package workflow

import (
	"aimc-go/aigc/core"
	"context"
	"testing"
)

func TestGraph_AsTool(t *testing.T) {
	g, _ := New("test").
		AddNode(&Node{
			ID:   "node1",
			Type: NodeTypeLLM,
			Config: NodeConfig{
				Task: core.TaskGeneralText,
			},
		}).
		SetStart("node1").
		Build()

	tool := g.AsTool("my_tool", "test tool")

	if tool.Name != "my_tool" {
		t.Errorf("expected 'my_tool', got %s", tool.Name)
	}

	if tool.Description != "test tool" {
		t.Errorf("expected 'test tool', got %s", tool.Description)
	}
}

func TestExecutor_ToolFunc(t *testing.T) {
	reg := core.NewRegistry()
	reg.Register(&mockModel{})

	router := core.NewRouter()
	router.Register(core.TaskGeneralText, "mock")

	client := core.NewClient(reg, router)
	executor := NewExecutor(client, nil)

	g, _ := New("test").
		AddNode(&Node{
			ID:   "node1",
			Type: NodeTypeLLM,
			Config: NodeConfig{
				Task:   core.TaskGeneralText,
				Prompt: "test",
			},
		}).
		SetStart("node1").
		Build()

	toolFunc := executor.ToolFunc(g)

	result, err := toolFunc(context.Background(), "{}")
	if err != nil {
		t.Fatal(err)
	}

	if result == "" {
		t.Error("expected result")
	}
}
