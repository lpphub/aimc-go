package server

import (
	"aimc-go/assistant/approval"
	"aimc-go/assistant/channel"
	"aimc-go/assistant/runtime"
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"time"
)

//go:embed sse.html
var staticFS embed.FS

// SSEServer SSE 服务
type SSEServer struct {
	rt  *runtime.Runtime
	hub *channel.SSEHub
}

// NewSSEServer 创建 SSE 服务
func NewSSEServer() (*SSEServer, error) {
	rt, err := NewRuntime()
	if err != nil {
		return nil, err
	}

	return &SSEServer{
		rt:  rt,
		hub: channel.NewSSEHub(),
	}, nil
}

// ChatRequest 聊天请求
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Query     string `json:"query"`
}

// Chat SSE 聊天接口
func (s *SSEServer) Chat(w http.ResponseWriter, r *http.Request) {
	// 设置 SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	ch, err := s.hub.Acquire(req.SessionID, w, flusher)
	if err != nil {
		// 会话忙，返回 409 Conflict
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// 异步运行 runtime
	go func() {
		defer func() {
			s.hub.Release(req.SessionID)
			ch.Close()
		}()

		err := s.rt.Run(ctx, ch, req.Query)
		if err != nil {
			ch.Write(channel.Chunk{
				Type:    channel.TypeError,
				Content: err.Error(),
			})
		}
	}()

	// 阻塞保持连接
	<-ctx.Done()
}

// ApprovalRequest 审批请求
type ApprovalRequest struct {
	SessionID  string `json:"session_id"`
	ApprovalID string `json:"approval_id"`
	Approved   bool   `json:"approved"`
	Reason     string `json:"reason,omitempty"`
}

// Approval 审批回调接口
func (s *SSEServer) Approval(w http.ResponseWriter, r *http.Request) {
	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	err := s.hub.SubmitApproval(req.SessionID, result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// SSEPage 测试页面
func (s *SSEServer) SSEPage(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("sse.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// Run 启动 SSE HTTP 服务
func (s *SSEServer) Run(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.SSEPage)
	mux.HandleFunc("/chat", s.Chat)
	mux.HandleFunc("/approval", s.Approval)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	return server.ListenAndServe()
}
