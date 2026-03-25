package aigc_test

import (
	"aimc-go/aigc/agent"
	"aimc-go/aigc/core"
	"aimc-go/aigc/workflow"
	"context"
	"testing"
)

type mockModel struct{}

func (m *mockModel) ID() core.ModelID { return "mock" }

func (m *mockModel) Generate(ctx context.Context, req *core.GenerateRequest) (*core.GenerateResponse, error) {
	return &core.GenerateResponse{Text: "mock response"}, nil
}

func TestIntegration_WorkflowAndAgent(t *testing.T) {
	// Setup core
	reg := core.NewRegistry()
	reg.Register(&mockModel{})

	router := core.NewRouter()
	router.Register(core.TaskGeneralText, "mock")

	client := core.NewClient(reg, router)

	// Create workflow
	g, _ := workflow.New("test-workflow").
		AddNode(&workflow.Node{
			ID:   "node1",
			Type: workflow.NodeTypeLLM,
			Config: workflow.NodeConfig{
				Task:   core.TaskGeneralText,
				Prompt: "test",
			},
		}).
		SetStart("node1").
		Build()

	// Execute workflow
	executor := workflow.NewExecutor(client, nil)
	result, err := executor.Run(context.Background(), g, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.Output == nil {
		t.Error("expected output")
	}

	// Create agent with workflow as tool
	a := agent.New("test-agent").
		WithModel("mock").
		WithSystemPrompt("You are helpful").
		AddGraph("workflow", g, executor).
		Build(client)

	agentResult, err := a.Run(context.Background(), "test input")
	if err != nil {
		t.Fatal(err)
	}

	if agentResult.FinalOutput == "" {
		t.Error("expected final output")
	}
}
