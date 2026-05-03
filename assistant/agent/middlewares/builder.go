package middlewares

import (
	"context"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

type ChainBuilder struct {
	ctx     context.Context
	model   model.BaseChatModel
	backend *local.Local

	items []adk.ChatModelAgentMiddleware
	err   error
}

func NewChainBuilder(ctx context.Context, model model.BaseChatModel, backend *local.Local) *ChainBuilder {
	return &ChainBuilder{ctx: ctx, model: model, backend: backend}
}

func (b *ChainBuilder) WithPatch() *ChainBuilder {
	if b.err != nil {
		return b
	}
	mw, err := newPatchMiddleware(b.ctx)
	if err != nil {
		b.err = err
		return b
	}
	b.items = append(b.items, mw)
	return b
}

func (b *ChainBuilder) WithSummarization(contextTokens int) *ChainBuilder {
	if b.err != nil {
		return b
	}
	mw, err := newSummarizationMiddleware(b.ctx, b.model, contextTokens)
	if err != nil {
		b.err = err
		return b
	}
	b.items = append(b.items, mw)
	return b
}

func (b *ChainBuilder) WithReduction() *ChainBuilder {
	if b.err != nil {
		return b
	}
	mw, err := newReductionMiddleware(b.ctx, b.backend)
	if err != nil {
		b.err = err
		return b
	}
	b.items = append(b.items, mw)
	return b
}

func (b *ChainBuilder) WithFilesystem() *ChainBuilder {
	if b.err != nil {
		return b
	}
	mw, err := newFilesystemMiddleware(b.ctx, b.backend)
	if err != nil {
		b.err = err
		return b
	}
	b.items = append(b.items, mw)
	return b
}

func (b *ChainBuilder) WithSkill(dir string) *ChainBuilder {
	if b.err != nil || dir == "" {
		return b
	}
	mw, err := newSkillMiddleware(b.ctx, b.backend, dir)
	if err != nil {
		b.err = err
		return b
	}
	b.items = append(b.items, mw)
	return b
}

func (b *ChainBuilder) WithPlanTask(dir string) *ChainBuilder {
	if b.err != nil || dir == "" {
		return b
	}
	mw, err := newPlanTaskMiddleware(b.ctx, b.backend, dir)
	if err != nil {
		b.err = err
		return b
	}
	b.items = append(b.items, mw)
	return b
}

func (b *ChainBuilder) WithToolRecovery() *ChainBuilder {
	if b.err != nil {
		return b
	}
	b.items = append(b.items, NewToolRecoveryMiddleware())
	return b
}

func (b *ChainBuilder) Build() ([]adk.ChatModelAgentMiddleware, error) {
	return b.items, b.err
}