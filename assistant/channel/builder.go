package channel

import (
	"aimc-go/assistant/approval"
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// CLIChannelBuilder CLI 场景的 Channel 构建器
type CLIChannelBuilder struct {
	scanner *bufio.Scanner
}

func NewCLIChannelBuilder(scanner *bufio.Scanner) *CLIChannelBuilder {
	if scanner == nil {
		panic("scanner cannot be nil")
	}
	return &CLIChannelBuilder{scanner: scanner}
}

// Build 创建 CLI Channel
func (b *CLIChannelBuilder) Build(sessionID string) *Channel {
	ch := NewChannel(sessionID, NewStdoutSink())

	ch.OnInput = func(ctx context.Context) (*InputEvent, error) {
		if !b.scanner.Scan() {
			return nil, fmt.Errorf("failed to read input: %w", b.scanner.Err())
		}
		response := strings.TrimSpace(b.scanner.Text())

		approved := response == "y" || response == "yes"
		return &InputEvent{
			Type: InputApproval,
			Data: &approval.Result{Approved: approved},
		}, nil
	}

	return ch
}

// SSEChannelBuilder SSE 场景的 Channel 构建器
type SSEChannelBuilder struct {
	channels sync.Map // sessionID -> *Channel
}

func NewSSEChannelBuilder() *SSEChannelBuilder {
	return &SSEChannelBuilder{}
}

// Build 创建 SSE Channel
func (b *SSEChannelBuilder) Build(sessionID string, w http.ResponseWriter, flusher http.Flusher) *Channel {
	ch := NewChannel(sessionID, NewSSESink(w, flusher))
	b.channels.Store(sessionID, ch)
	return ch
}

// SubmitApproval 提交审批结果（供 HTTP handler 调用）
func (b *SSEChannelBuilder) SubmitApproval(sessionID string, result *approval.Result) error {
	c, ok := b.channels.Load(sessionID)
	if !ok {
		return fmt.Errorf("channel not found: %s", sessionID)
	}

	ch := c.(*Channel)
	ch.Input <- InputEvent{
		Type: InputApproval,
		Data: result,
	}
	return nil
}

// Remove 移除 channel（清理）
func (b *SSEChannelBuilder) Remove(sessionID string) {
	b.channels.Delete(sessionID)
}