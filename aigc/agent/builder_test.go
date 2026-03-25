package agent

import (
	"aimc-go/aigc/core"
	"context"
	"testing"
)

func TestBuilder(t *testing.T) {
	b := New("test-agent").
		WithModel("openai-gpt4o").
		WithSystemPrompt("You are helpful").
		WithMaxTurns(10).
		AddTool(core.ToolDefinition{Name: "test"}, func(ctx context.Context, args string) (string, error) {
			return "result", nil
		})

	if b.config.Name != "test-agent" {
		t.Errorf("expected 'test-agent', got %s", b.config.Name)
	}

	if b.config.Model != "openai-gpt4o" {
		t.Errorf("expected 'openai-gpt4o', got %s", b.config.Model)
	}

	if b.config.MaxTurns != 10 {
		t.Errorf("expected 10, got %d", b.config.MaxTurns)
	}

	if len(b.config.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(b.config.Tools))
	}
}
