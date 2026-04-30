package server

import (
	"aimc-go/app/modules/core"
	"aimc-go/assistant/runtime"
	"aimc-go/assistant/session"
	"aimc-go/assistant/types"
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var _ core.Module = (*SSEModule)(nil)

//go:embed sse.html
var staticFS embed.FS

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

	sess := session.New(sessionID, session.NewSSETransport(ctx, w, flusher))
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
	transport, ok := sess.Transport.(*session.SSETransport)
	if !ok {
		return fmt.Errorf("session %s transport does not support Submit", sessionID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return transport.Submit(ctx, session.InputEvent{
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

type SSEModule struct {
	rt  *runtime.Runtime
	hub *SSEHub
}

func NewSSE() (*SSEModule, error) {
	rt, err := NewRuntime()
	if err != nil {
		return nil, err
	}

	return &SSEModule{
		rt:  rt,
		hub: NewSSEHub(),
	}, nil
}

func (m *SSEModule) Routes(r *gin.RouterGroup) {
	assistant := r.Group("/assistant")
	assistant.GET("", m.ssePage)
	assistant.POST("/chat", m.chat)
	assistant.POST("/approval", m.approval)
}

func (m *SSEModule) chat(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	var req struct {
		SessionID string `json:"session_id"`
		Query     string `json:"query"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.AbortWithError(400, err)
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		_ = c.AbortWithError(500, errors.New("streaming unsupported"))
		return
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	sess, err := m.hub.Acquire(ctx, req.SessionID, c.Writer, flusher)
	if err != nil {
		_ = c.AbortWithError(409, err)
		return
	}

	go func() {
		defer cancel()
		defer m.hub.Release(req.SessionID)

		err := m.rt.Run(ctx, sess, req.Query)
		if err != nil {
			_ = sess.Emit(session.Event{
				Type:    session.TypeMessage,
				Content: err.Error(),
			})
		}
	}()

	<-ctx.Done()
}

func (m *SSEModule) approval(c *gin.Context) {
	var req struct {
		SessionID  string `json:"session_id"`
		ApprovalID string `json:"approval_id"`
		Approved   bool   `json:"approved"`
		Reason     string `json:"reason,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.AbortWithError(400, err)
		return
	}

	result := &types.ApprovalResult{
		ApprovalID: req.ApprovalID,
		Approved:   req.Approved,
	}
	if req.Reason != "" {
		result.DisapproveReason = &req.Reason
	}

	err := m.hub.SubmitApproval(req.SessionID, result)
	if err != nil {
		_ = c.AbortWithError(404, err)
		return
	}

	c.JSON(200, gin.H{"status": "ok"})
}

func (m *SSEModule) ssePage(c *gin.Context) {
	data, err := staticFS.ReadFile("sse.html")
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Data(200, "text/html; charset=utf-8", data)
}
