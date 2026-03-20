package aigc

import (
	"aimc-go/aigc/models"
	"context"
	"testing"
)

func TestGemini_Generate(t *testing.T) {
	registry := NewRegistry()

	registry.Register(&models.OpenAI{})
	registry.Register(&models.Gemini{})

	ai := NewClient(registry)

	text, _ := ai.Text(context.Background(), "gemini", "hello world")
	t.Log(text)
}
