package agent

import (
	"aimc-go/aigc/core"
	"aimc-go/aigc/workflow"
)

type Builder struct {
	config *AgentConfig
}

func New(name string) *Builder {
	return &Builder{config: &AgentConfig{
		Name:      name,
		Tools:     []core.ToolDefinition{},
		ToolFuncs: map[string]core.ToolFunc{},
		MaxTurns:  10,
	}}
}

func (b *Builder) WithModel(model core.ModelID) *Builder {
	b.config.Model = model
	return b
}

func (b *Builder) WithSystemPrompt(prompt string) *Builder {
	b.config.SystemPrompt = prompt
	return b
}

func (b *Builder) WithMaxTurns(n int) *Builder {
	b.config.MaxTurns = n
	return b
}

func (b *Builder) AddTool(def core.ToolDefinition, fn core.ToolFunc) *Builder {
	b.config.Tools = append(b.config.Tools, def)
	b.config.ToolFuncs[def.Name] = fn
	return b
}

func (b *Builder) AddGraph(name string, g *workflow.Graph, executor *workflow.Executor) *Builder {
	tool := g.AsTool(name, "Workflow: "+name)
	toolFunc := executor.ToolFunc(g)
	b.config.Tools = append(b.config.Tools, tool)
	b.config.ToolFuncs[name] = toolFunc
	return b
}

func (b *Builder) Build(client *core.Client) Agent {
	return &ReactAgent{
		config: b.config,
		client: client,
	}
}
