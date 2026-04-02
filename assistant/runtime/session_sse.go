package runtime

import (
	"aimc-go/assistant/approval"
	"aimc-go/assistant/sink"
	"context"
	"fmt"
	"net/http"
	"sync"
)

// SSESessionBuilder SSE 场景的 SessionBuilder
type SSESessionBuilder struct {
	sessions sync.Map // sessionID -> *Session
}

// NewSSESessionBuilder 创建 SSE SessionBuilder
func NewSSESessionBuilder() *SSESessionBuilder {
	return &SSESessionBuilder{}
}

// Build 创建 SSE Session
func (b *SSESessionBuilder) Build(ctx context.Context, sessionID string, w http.ResponseWriter, flusher http.Flusher) (*Session, error) {
	session := NewSession(ctx, sessionID, sink.NewSSESink(w, flusher))
	// 不注册 OnInput，使用 InputChan

	b.sessions.Store(sessionID, session)
	return session, nil
}

// SubmitApproval 提交审批结果（供 HTTP handler 调用）
func (b *SSESessionBuilder) SubmitApproval(sessionID string, result *approval.ApprovalResult) error {
	sess, ok := b.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session := sess.(*Session)
	session.InputChan <- InputEvent{
		Type: InputApproval,
		Data: result,
	}
	return nil
}

// GetSession 获取已存在的 session（用于审批回调）
func (b *SSESessionBuilder) GetSession(sessionID string) (*Session, error) {
	sess, ok := b.sessions.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return sess.(*Session), nil
}

// RemoveSession 移除 session（清理）
func (b *SSESessionBuilder) RemoveSession(sessionID string) {
	b.sessions.Delete(sessionID)
}