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

func TestClient_MarketingCopy(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockModel{id: "mock-openai"})

	router := NewRouter()
	router.SetDefault(TaskMarketingCopy, "mock-openai")

	client := NewClient(reg, router)

	resp, err := client.MarketingCopy(context.Background(), "推广运动鞋")
	if err != nil {
		t.Fatal(err)
	}

	if resp.Text == "" {
		t.Error("expected non-empty response")
	}

	expectedText := "mock: 你是一位资深营销文案专家。请根据以下需求撰写营销文案：\n\n需求：推广运动鞋\n\n要求：\n- 语言简洁有力\n- 突出卖点\n- 适合社交媒体传播"
	if resp.Text != expectedText {
		t.Errorf("unexpected response: %s", resp.Text)
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
	expectedText := "mock: 你是一位资深营销文案专家。请根据以下需求撰写营销文案：\n\n需求：test\n\n要求：\n- 语言简洁有力\n- 突出卖点\n- 适合社交媒体传播"
	if resp.Text != expectedText {
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
