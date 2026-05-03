package middlewares

import (
	"context"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/patchtoolcalls"
	"github.com/cloudwego/eino/adk/middlewares/plantask"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/components/model"
)

func newPatchMiddleware(ctx context.Context) (adk.ChatModelAgentMiddleware, error) {
	return patchtoolcalls.New(ctx, &patchtoolcalls.Config{})
}

func newSummarizationMiddleware(ctx context.Context, model model.BaseChatModel, contextTokens int) (adk.ChatModelAgentMiddleware, error) {
	return summarization.New(ctx, &summarization.Config{
		Model:   model,
		Trigger: &summarization.TriggerCondition{ContextTokens: contextTokens},
	})
}

func newReductionMiddleware(ctx context.Context, backend *local.Local) (adk.ChatModelAgentMiddleware, error) {
	return reduction.New(ctx, &reduction.Config{Backend: backend})
}

func newFilesystemMiddleware(ctx context.Context, backend *local.Local) (adk.ChatModelAgentMiddleware, error) {
	return filesystem.New(ctx, &filesystem.MiddlewareConfig{
		Backend:        backend,
		StreamingShell: backend,
	})
}

func newSkillMiddleware(ctx context.Context, backend *local.Local, dir string) (adk.ChatModelAgentMiddleware, error) {
	skillBackend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
		Backend: backend,
		BaseDir: dir,
	})
	if err != nil {
		return nil, err
	}
	return skill.NewMiddleware(ctx, &skill.Config{Backend: skillBackend})
}

func newPlanTaskMiddleware(ctx context.Context, backend *local.Local, dir string) (adk.ChatModelAgentMiddleware, error) {
	return plantask.New(ctx, &plantask.Config{
		Backend: &planTaskBackend{Local: backend},
		BaseDir: dir,
	})
}

type planTaskBackend struct {
	*local.Local
}

func (b *planTaskBackend) Delete(ctx context.Context, req *plantask.DeleteRequest) error {
	return nil
}