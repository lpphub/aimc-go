package agent

import (
	"aimc-go/aigc/core"
	"context"
)

type Agent interface {
	ID() string
	Run(ctx context.Context, input string) (*AgentResult, error)
	RunStream(ctx context.Context, input string) (<-chan AgentEvent, error)
}

type AgentConfig struct {
	Name         string
	Description  string
	SystemPrompt string
	Model        core.ModelID
	Tools        []core.ToolDefinition
	ToolFuncs    map[string]core.ToolFunc
	MaxTurns     int
}

type AgentResult struct {
	FinalOutput string
	Steps       []AgentStep
	ToolCalls   []core.ToolCall
}

type AgentStep struct {
	Role     string
	Content  string
	ToolCall *core.ToolCall
}

type AgentEvent struct {
	Type    string
	Content string
	Done    bool
}
