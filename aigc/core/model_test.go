package core

import (
	"context"
	"testing"
)

type mockModel struct {
	id ModelID
}

func (m *mockModel) ID() ModelID { return m.id }

func (m *mockModel) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{Text: "mock"}, nil
}

func TestModel(t *testing.T) {
	m := &mockModel{id: "test"}
	if m.ID() != "test" {
		t.Errorf("expected 'test', got %s", m.ID())
	}

	resp, err := m.Generate(context.Background(), &GenerateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "mock" {
		t.Errorf("expected 'mock', got %s", resp.Text)
	}
}
