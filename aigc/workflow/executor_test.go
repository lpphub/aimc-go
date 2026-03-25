package workflow

import (
	"aimc-go/aigc/core"
	"context"
	"testing"
)

type mockModel struct{}

func (m *mockModel) ID() core.ModelID { return "mock" }

func (m *mockModel) Generate(ctx context.Context, req *core.GenerateRequest) (*core.GenerateResponse, error) {
	return &core.GenerateResponse{Text: "result"}, nil
}

func TestExecutor(t *testing.T) {
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

	result, err := executor.Run(context.Background(), g, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.Output == nil {
		t.Error("expected output")
	}
}
