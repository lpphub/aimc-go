package x

import (
	"aimc-go/aigc"
	"context"
	"testing"
)

func TestInit_Defaults(t *testing.T) {
	err := Init(context.Background(), Config{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerate_BeforeInit(t *testing.T) {
	// Reset client
	client = nil
	_, err := Generate(context.Background(), &aigc.GenerateRequest{
		Task:   aigc.TaskMarketingCopy,
		Prompt: "test",
	})
	if err == nil {
		t.Error("expected error before Init")
	}
}
