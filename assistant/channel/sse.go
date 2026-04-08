package channel

import (
	"aimc-go/assistant/approval"
	"context"
	"fmt"
	"net/http"
	"sync"
)

// SSEHub 管理 SSE 场景的 Channel
type SSEHub struct {
	mu       sync.RWMutex
	channels map[string]*Channel // sessionID -> *Channel
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		channels: make(map[string]*Channel),
	}
}

// Acquire 获取或创建 Channel，如果会话忙则返回错误
func (h *SSEHub) Acquire(ctx context.Context, sessionID string, w http.ResponseWriter, flusher http.Flusher) (*Channel, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.channels[sessionID]; ok {
		return nil, fmt.Errorf("session %s is busy", sessionID)
	}

	ch := NewChannel(sessionID, NewSSEWriter(ctx, w, flusher))
	h.channels[sessionID] = ch
	return ch, nil
}

// SubmitApproval 提交审批结果
func (h *SSEHub) SubmitApproval(sessionID string, result *approval.Result) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ch, ok := h.channels[sessionID]
	if !ok {
		return fmt.Errorf("channel not found: %s", sessionID)
	}

	// 使用 select 防止向已关闭的 channel 发送导致 panic
	select {
	case ch.Input <- InputEvent{
		Type: InputApproval,
		Data: result,
	}:
		return nil
	default:
		return fmt.Errorf("channel closed or full: %s", sessionID)
	}
}

// Release 释放 Channel
func (h *SSEHub) Release(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.channels, sessionID)
}