package core

import (
	"context"
	"testing"
)

func TestClient_Generate(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockModel{id: "model-a"})

	router := NewRouter()
	router.Register(TaskMarketingCopy, "model-a")

	client := NewClient(reg, router)

	resp, err := client.Generate(context.Background(), &GenerateRequest{
		Task:   TaskMarketingCopy,
		Prompt: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "mock" {
		t.Errorf("expected 'mock', got %s", resp.Text)
	}
}

func TestClient_Generate_NoModel(t *testing.T) {
	reg := NewRegistry()
	router := NewRouter()
	client := NewClient(reg, router)

	_, err := client.Generate(context.Background(), &GenerateRequest{
		Task: TaskGeneralText,
	})
	if err == nil {
		t.Error("expected error")
	}
}
