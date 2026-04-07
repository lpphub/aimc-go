package server

import (
	"aimc-go/app/modules/core"
	"aimc-go/assistant/approval"
	"aimc-go/assistant/channel"
	"aimc-go/assistant/runtime"
	"context"
	"embed"

	"github.com/gin-gonic/gin"
)

var _ core.Module = (*SSEModule)(nil)

//go:embed sse.html
var staticFS embed.FS

type SSEModule struct {
	rt  *runtime.Runtime
	hub *channel.SSEHub
}

func NewSSE() (*SSEModule, error) {
	rt, err := NewRuntime()
	if err != nil {
		return nil, err
	}

	return &SSEModule{
		rt:  rt,
		hub: channel.NewSSEHub(),
	}, nil
}

func (m *SSEModule) Routes(r *gin.RouterGroup) {
	assistant := r.Group("/assistant")
	assistant.GET("", m.ssePage)
	assistant.POST("/chat", m.chat)
	assistant.POST("/approval", m.approval)
}

func (m *SSEModule) chat(c *gin.Context) {
	// 设置 SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	var req struct {
		SessionID string `json:"session_id"`
		Query     string `json:"query"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithError(400, err)
		return
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ch, err := m.hub.Acquire(req.SessionID, c.Writer, c.Writer)
	if err != nil {
		c.AbortWithError(409, err)
		return
	}

	// 异步运行 runtime
	go func() {
		defer cancel()
		defer ch.Close()
		defer m.hub.Release(req.SessionID)

		err := m.rt.Run(ctx, ch, req.Query)
		if err != nil {
			ch.Write(channel.Chunk{
				Type:    channel.TypeError,
				Content: err.Error(),
			})
		}
	}()

	// 阻塞保持连接（客户端断开或任务完成都会退出）
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
		c.AbortWithError(400, err)
		return
	}

	result := &approval.Result{
		ApprovalID:       req.ApprovalID,
		Approved:         req.Approved,
		DisapproveReason: nil,
	}
	if req.Reason != "" {
		result.DisapproveReason = &req.Reason
	}

	err := m.hub.SubmitApproval(req.SessionID, result)
	if err != nil {
		c.AbortWithError(404, err)
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
