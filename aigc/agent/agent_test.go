package agent

import (
	"aimc-go/aigc/core"
	"testing"
)

func TestAgentResult(t *testing.T) {
	result := &AgentResult{
		FinalOutput: "test output",
		Steps: []AgentStep{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
		},
	}

	if result.FinalOutput != "test output" {
		t.Errorf("expected 'test output', got %s", result.FinalOutput)
	}

	if len(result.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(result.Steps))
	}
}

func TestAgentConfig(t *testing.T) {
	config := &AgentConfig{
		Name:         "test",
		SystemPrompt: "You are helpful",
		Model:        "openai-gpt4o",
		MaxTurns:     10,
		Tools:        []core.ToolDefinition{},
		ToolFuncs:    map[string]core.ToolFunc{},
	}

	if config.Name != "test" {
		t.Errorf("expected 'test', got %s", config.Name)
	}
}
