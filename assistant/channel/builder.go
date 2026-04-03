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
	mu       sync.RWMutex
	channels map[string]*Channel // sessionID -> *Channel
}

func NewSSEChannelBuilder() *SSEChannelBuilder {
	return &SSEChannelBuilder{
		channels: make(map[string]*Channel),
	}
}

// Build 创建 SSE Channel，如果会话忙则返回错误
func (b *SSEChannelBuilder) Build(sessionID string, w http.ResponseWriter, flusher http.Flusher) (*Channel, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 检查是否已存在（会话忙）
	if _, ok := b.channels[sessionID]; ok {
		return nil, fmt.Errorf("session %s is busy", sessionID)
	}

	ch := NewChannel(sessionID, NewSSESink(w, flusher))
	b.channels[sessionID] = ch
	return ch, nil
}

// SubmitApproval 提交审批结果（供 HTTP handler 调用）
func (b *SSEChannelBuilder) SubmitApproval(sessionID string, result *approval.Result) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	ch, ok := b.channels[sessionID]
	if !ok {
		return fmt.Errorf("channel not found: %s", sessionID)
	}

	ch.Input <- InputEvent{
		Type: InputApproval,
		Data: result,
	}
	return nil
}

// Remove 移除 channel（清理）
func (b *SSEChannelBuilder) Remove(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.channels, sessionID)
}