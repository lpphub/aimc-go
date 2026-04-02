package api

import (
	"aimc-go/assistant/approval"
	"aimc-go/assistant/runtime"
	"aimc-go/assistant/sink"
	"encoding/json"
	"net/http"
)

// Handler HTTP handler
type Handler struct {
	rt             *runtime.Runtime
	sessionBuilder *runtime.SSESessionBuilder
}

// NewHandler 创建 Handler
func NewHandler(rt *runtime.Runtime) *Handler {
	return &Handler{
		rt:             rt,
		sessionBuilder: runtime.NewSSESessionBuilder(),
	}
}

// ChatRequest 聊天请求
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Query     string `json:"query"`
}

// Chat SSE 聊天接口
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
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

	session, err := h.sessionBuilder.Build(ctx, req.SessionID, w, flusher)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 异步运行 runtime
	go func() {
		err := h.rt.Run(ctx, session, req.Query)
		if err != nil {
			session.Emit(sink.Chunk{
				Type:    sink.TypeError,
				Content: err.Error(),
			})
		}
		// 运行结束后清理 session
		h.sessionBuilder.RemoveSession(req.SessionID)
		session.Close()
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
func (h *Handler) Approval(w http.ResponseWriter, r *http.Request) {
	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := &approval.ApprovalResult{
		Approved:         req.Approved,
		DisapproveReason: nil,
	}
	if req.Reason != "" {
		result.DisapproveReason = &req.Reason
	}

	err := h.sessionBuilder.SubmitApproval(req.SessionID, result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}