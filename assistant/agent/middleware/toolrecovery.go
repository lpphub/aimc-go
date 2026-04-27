package middleware

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// toolRecoveryMiddleware 将可恢复的工具错误转为文本回送 LLM，
// 避免工具执行失败直接中断对话。LLM 收到错误文本后可自主重试或调整参数。
type toolRecoveryMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func (m *toolRecoveryMiddleware) WrapInvokableToolCall(_ context.Context, endpoint adk.InvokableToolCallEndpoint, _ *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		result, err := endpoint(ctx, args, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return "", err
			}
			if isFatalError(err) {
				return "", err
			}
			return fmt.Sprintf("[tool error] %v", err), nil
		}
		return result, nil
	}, nil
}

func (m *toolRecoveryMiddleware) WrapStreamableToolCall(_ context.Context, endpoint adk.StreamableToolCallEndpoint, _ *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		sr, err := endpoint(ctx, args, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return nil, err
			}
			if isFatalError(err) {
				return nil, err
			}
			return singleChunkReader(fmt.Sprintf("[tool error] %v", err)), nil
		}

		r, w := schema.Pipe[string](64)
		go func() {
			defer w.Close()
			for {
				chunk, err := sr.Recv()
				if errors.Is(err, io.EOF) {
					return
				}
				if err != nil {
					if isFatalError(err) {
						w.Close()
						return
					}
					_ = w.Send(fmt.Sprintf("\n[tool error] %v", err), nil)
					return
				}
				_ = w.Send(chunk, nil)
			}
		}()
		return r, nil
	}, nil
}

// isFatalError 判断错误是否应直接终止而非重试。
func isFatalError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
