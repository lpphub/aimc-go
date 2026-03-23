package aigc

import "testing"

func TestRouter_Resolve_ExplicitOverride(t *testing.T) {
	r := NewRouter()
	r.SetDefault(TaskMarketingCopy, "openai-gpt4o")

	req := &GenerateRequest{
		Task:  TaskMarketingCopy,
		Model: "gemini-2.0-flash",
	}

	got := r.Resolve(req)
	if got != "gemini-2.0-flash" {
		t.Errorf("expected gemini-2.0-flash, got %s", got)
	}
}

func TestRouter_Resolve_Default(t *testing.T) {
	r := NewRouter()
	r.SetDefault(TaskMarketingCopy, "openai-gpt4o")

	req := &GenerateRequest{Task: TaskMarketingCopy}

	got := r.Resolve(req)
	if got != "openai-gpt4o" {
		t.Errorf("expected openai-gpt4o, got %s", got)
	}
}

func TestRouter_Resolve_NoDefault(t *testing.T) {
	r := NewRouter()

	req := &GenerateRequest{Task: TaskGeneralText}

	got := r.Resolve(req)
	if got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}
