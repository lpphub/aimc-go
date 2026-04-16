package server

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/runtime"
	"aimc-go/assistant/store"
	"context"
)

// NewRuntime 创建 Runtime（公共方法）
func NewRuntime() (*runtime.Runtime, error) {
	ctx := context.Background()

	assistantAgent, err := agent.New(ctx,
		agent.WithProjectRoot("/Users/lsk/Projects/eino-demo"),
		agent.WithSkillDir("/Users/lsk/Projects/eino-demo/ext/skills"),
		agent.WithPlanTaskDir("/Users/lsk/Projects/aimc-go/docs/plans"),
	)
	if err != nil {
		return nil, err
	}

	jsonlStore := store.NewJSONLStore("./data/conversations")

	return runtime.New(assistantAgent, runtime.WithStore(jsonlStore))
}
