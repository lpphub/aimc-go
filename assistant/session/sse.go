package session

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

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
	sess.Input = make(chan InputEvent, 1)
	sess.closed = make(chan struct{})
	
	h.sessions[sessionID] = sess
	return sess, nil
}

// SubmitApproval 提交审批结果，阻塞等待 session 接收
func (h *SSEHub) SubmitApproval(sessionID string, result *approval.Result) error {
	h.mu.RLock()
	sess, ok := h.sessions[sessionID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	select {
	case sess.Input <- InputEvent{
		Type: InputApproval,
		Data: result,
	}:
		return nil
	case <-sess.Closed():
		return fmt.Errorf("session closed: %s", sessionID)
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for session to receive approval")
	}
}

// Release 释放 Session
func (h *SSEHub) Release(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.sessions, sessionID)
}
