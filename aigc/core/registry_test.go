package core

import "testing"

func TestRegistry(t *testing.T) {
	reg := NewRegistry()

	m := &mockModel{id: "test-model"}
	reg.Register(m)

	got, err := reg.Get("test-model")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID() != "test-model" {
		t.Errorf("expected 'test-model', got %s", got.ID())
	}

	_, err = reg.Get("not-exist")
	if err == nil {
		t.Error("expected error for not-exist")
	}

	ids := reg.List()
	if len(ids) != 1 || ids[0] != "test-model" {
		t.Errorf("unexpected list: %v", ids)
	}
}
