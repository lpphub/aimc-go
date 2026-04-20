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
	sessions sync.Map // sessionID -> *Session (无锁并发访问)
}

func NewSSEHub() *SSEHub {
	return &SSEHub{}
}

// Acquire 获取或创建 Session，如果会话忙则返回错误
func (h *SSEHub) Acquire(ctx context.Context, sessionID string, w http.ResponseWriter, flusher http.Flusher) (*Session, error) {
	sess := New(sessionID, NewSSEWriter(ctx, w, flusher))
	sess.Input = make(chan InputEvent, 1)
	sess.closed = make(chan struct{})

	// sync.Map 的 LoadOrStore 是原子操作
	actual, loaded := h.sessions.LoadOrStore(sessionID, sess)
	if loaded {
		return nil, fmt.Errorf("session %s is busy", sessionID)
	}

	return actual.(*Session), nil
}

// SubmitApproval 提交审批结果，阻塞等待 session 接收
func (h *SSEHub) SubmitApproval(sessionID string, result *approval.Result) error {
	sess, ok := h.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	s := sess.(*Session)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	select {
	case s.Input <- InputEvent{
		Type: InputApproval,
		Data: result,
	}:
		return nil
	case <-s.Closed():
		return fmt.Errorf("session closed: %s", sessionID)
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for session to receive approval")
	}
}

// Release 释放 Session
func (h *SSEHub) Release(sessionID string) {
	h.sessions.Delete(sessionID)
}
