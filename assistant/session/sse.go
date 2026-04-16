package session

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"aimc-go/assistant/approval"
)

// SSEHub 管理 SSE 场景的 Session
type SSEHub struct {
	mu       sync.RWMutex
	sessions map[string]*Session // sessionID -> *Session
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		sessions: make(map[string]*Session),
	}
}

// Acquire 获取或创建 Session，如果会话忙则返回错误
func (h *SSEHub) Acquire(ctx context.Context, sessionID string, w http.ResponseWriter, flusher http.Flusher) (*Session, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.sessions[sessionID]; ok {
		return nil, fmt.Errorf("session %s is busy", sessionID)
	}

	sess := New(sessionID, NewSSEWriter(ctx, w, flusher))
	h.sessions[sessionID] = sess
	return sess, nil
}

// SubmitApproval 提交审批结果
func (h *SSEHub) SubmitApproval(sessionID string, result *approval.Result) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sess, ok := h.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// 使用 select 防止向已关闭的 channel 发送导致 panic
	select {
	case sess.Input <- InputEvent{
		Type: InputApproval,
		Data: result,
	}:
		return nil
	default:
		return fmt.Errorf("session closed or full: %s", sessionID)
	}
}

// Release 释放 Session
func (h *SSEHub) Release(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.sessions, sessionID)
}