package tools

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

type ChainBuilder struct {
	ctx   context.Context
	model model.BaseChatModel

	tools []tool.BaseTool
	err   error
}

func NewChainBuilder(ctx context.Context, model model.BaseChatModel) *ChainBuilder {
	return &ChainBuilder{ctx: ctx, model: model}
}

func (b *ChainBuilder) Build() ([]tool.BaseTool, error) {
	return b.tools, b.err
}

func (b *ChainBuilder) WithRAG() *ChainBuilder {
	if b.err != nil {
		return b
	}
	t, err := BuildRAGTool(b.ctx, b.model)
	if err != nil {
		b.err = err
		return b
	}
	b.tools = append(b.tools, t)
	return b
}

func (b *ChainBuilder) WithTime() *ChainBuilder {
	if b.err != nil {
		return b
	}
	t, err := NewTimeTool()
	if err != nil {
		b.err = err
		return b
	}
	b.tools = append(b.tools, t)
	return b
}
