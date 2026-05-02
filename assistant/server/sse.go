package server

import (
	"aimc-go/assistant/session"
	"aimc-go/assistant/types"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEHub 管理 SSE 场景的 Session
type SSEHub struct {
	sessions sync.Map // sessionID -> *session.Session
}

func NewSSEHub() *SSEHub {
	return &SSEHub{}
}

// Acquire 获取或创建 Session，如果会话忙则返回错误
func (h *SSEHub) Acquire(ctx context.Context, sessionID string, w http.ResponseWriter, flusher http.Flusher) (*session.Session, error) {
	if _, exists := h.sessions.Load(sessionID); exists {
		return nil, fmt.Errorf("session %s is busy", sessionID)
	}

	sess := session.New(sessionID, session.NewSSEEndpoint(ctx, w, flusher))
	actual, loaded := h.sessions.LoadOrStore(sessionID, sess)
	if loaded {
		return nil, fmt.Errorf("session %s is busy", sessionID)
	}
	return actual.(*session.Session), nil
}

// SubmitApproval 提交审批结果
func (h *SSEHub) SubmitApproval(sessionID string, result *types.ApprovalResult) error {
	val, ok := h.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	sess := val.(*session.Session)
	sink, ok := sess.Endpoint.(session.InputSink)
	if !ok {
		return fmt.Errorf("session %s endpoint does not support input", sessionID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return sink.Accept(ctx, session.InputEvent{
		Type: session.InputApproval,
		Data: result,
	})
}

// Release 释放 Session
func (h *SSEHub) Release(sessionID string) {
	val, ok := h.sessions.LoadAndDelete(sessionID)
	if ok {
		val.(*session.Session).Close()
	}
}
