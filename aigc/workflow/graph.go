package workflow

import (
	"aimc-go/aigc/core"
	"context"
	"fmt"
)

type NodeType string

const (
	NodeTypeLLM      NodeType = "llm"
	NodeTypeTool     NodeType = "tool"
	NodeTypeBranch   NodeType = "branch"
	NodeTypeParallel NodeType = "parallel"
	NodeTypeSubGraph NodeType = "subgraph"
)

type Node struct {
	ID     string
	Type   NodeType
	Config NodeConfig
}

type NodeConfig struct {
	Task      core.TaskType
	Prompt    string
	ToolName  string
	ToolFunc  core.ToolFunc
	Condition func(ctx context.Context, input any) string
	Graph     *Graph
	Params    map[string]any
}

type Edge struct {
	From string
	To   string
}

type Graph struct {
	Name  string
	Nodes map[string]*Node
	Edges []Edge
	Start string
}

type Builder struct {
	g *Graph
}

func New(name string) *Builder {
	return &Builder{g: &Graph{
		Name:  name,
		Nodes: make(map[string]*Node),
	}}
}

func (b *Builder) AddNode(node *Node) *Builder {
	b.g.Nodes[node.ID] = node
	return b
}

func (b *Builder) Connect(from, to string) *Builder {
	b.g.Edges = append(b.g.Edges, Edge{From: from, To: to})
	return b
}

func (b *Builder) SetStart(nodeID string) *Builder {
	b.g.Start = nodeID
	return b
}

func (b *Builder) Build() (*Graph, error) {
	if b.g.Start == "" {
		return nil, fmt.Errorf("start node not set")
	}
	if _, ok := b.g.Nodes[b.g.Start]; !ok {
		return nil, fmt.Errorf("start node %s not found", b.g.Start)
	}
	return b.g, nil
}
