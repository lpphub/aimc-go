package core

import "testing"

func TestRouter(t *testing.T) {
	r := NewRouter()
	r.Register(TaskMarketingCopy, "openai-gpt4o")

	req := &GenerateRequest{
		Task:  TaskMarketingCopy,
		Model: "gemini-2.0-flash",
	}

	got := r.Resolve(req)
	if got != "gemini-2.0-flash" {
		t.Errorf("expected 'gemini-2.0-flash', got %s", got)
	}

	req2 := &GenerateRequest{Task: TaskMarketingCopy}
	got2 := r.Resolve(req2)
	if got2 != "openai-gpt4o" {
		t.Errorf("expected 'openai-gpt4o', got %s", got2)
	}
}
