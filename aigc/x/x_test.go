package x

import (
	"aimc-go/aigc/core"
	"context"
	"testing"
)

func TestInit_Defaults(t *testing.T) {
	Init()
}

func TestGenerate_BeforeInit(t *testing.T) {
	// Reset client
	client = nil
	_, err := Generate(context.Background(), &core.GenerateRequest{
		Task:   core.TaskMarketingCopy,
		Prompt: "test",
	})
	if err == nil {
		t.Error("expected error before Init")
	}
}
