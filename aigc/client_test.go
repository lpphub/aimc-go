package aigc

import (
	"context"
	"testing"
)

type mockModel struct {
	id ModelID
}

func (m *mockModel) ID() ModelID { return m.id }

func (m *mockModel) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{Text: "mock: " + req.Prompt}, nil
}

func TestClient_Generate_RawPrompt(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockModel{id: "model-a"})

	router := NewRouter()
	router.SetDefault(TaskMarketingCopy, "model-a")

	client := NewClient(reg, router)

	resp, err := client.Generate(context.Background(), &GenerateRequest{
		Task:   TaskMarketingCopy,
		Prompt: "raw input",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "mock: raw input" {
		t.Errorf("expected raw prompt, got: %s", resp.Text)
	}
}

func TestClient_Generate_ModelOverride(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockModel{id: "model-a"})
	reg.Register(&mockModel{id: "model-b"})

	router := NewRouter()
	router.SetDefault(TaskMarketingCopy, "model-a")

	client := NewClient(reg, router)

	resp, err := client.Generate(context.Background(), &GenerateRequest{
		Task:   TaskMarketingCopy,
		Model:  "model-b",
		Prompt: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "mock: test" {
		t.Errorf("unexpected: %s", resp.Text)
	}
}

func TestClient_Generate_NoModel(t *testing.T) {
	reg := NewRegistry()
	router := NewRouter()

	client := NewClient(reg, router)

	_, err := client.Generate(context.Background(), &GenerateRequest{
		Task:   TaskGeneralText,
		Prompt: "test",
	})
	if err == nil {
		t.Error("expected error for unresolved model")
	}
}
