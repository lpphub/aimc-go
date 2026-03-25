package core

import (
	"context"
	"testing"
)

func TestToolFunc(t *testing.T) {
	fn := func(ctx context.Context, args string) (string, error) {
		return "result: " + args, nil
	}

	result, err := fn(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if result != "result: test" {
		t.Errorf("expected 'result: test', got %s", result)
	}
}
