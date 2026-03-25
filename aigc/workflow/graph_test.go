package workflow

import (
	"aimc-go/aigc/core"
	"testing"
)

func TestGraph(t *testing.T) {
	g, err := New("test").
		AddNode(&Node{
			ID:   "node1",
			Type: NodeTypeLLM,
			Config: NodeConfig{
				Task: core.TaskGeneralText,
			},
		}).
		AddNode(&Node{
			ID:   "node2",
			Type: NodeTypeTool,
			Config: NodeConfig{
				ToolName: "test-tool",
			},
		}).
		Connect("node1", "node2").
		SetStart("node1").
		Build()

	if err != nil {
		t.Fatal(err)
	}

	if g.Name != "test" {
		t.Errorf("expected 'test', got %s", g.Name)
	}

	if len(g.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(g.Nodes))
	}

	if len(g.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(g.Edges))
	}

	if g.Start != "node1" {
		t.Errorf("expected start 'node1', got %s", g.Start)
	}
}
