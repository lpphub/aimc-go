package channel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// ChunkType 输出片段类型
type ChunkType string

const (
	TypeAssistant   ChunkType = "assistant"
	TypeToolCall    ChunkType = "tool_call"
	TypeToolResult  ChunkType = "tool_result"
	TypeMessage     ChunkType = "message"
	TypeApproval    ChunkType = "approval"        // 审批请求
	TypeApprovalRes ChunkType = "approval_result" // 审批结果
	TypeError       ChunkType = "error"           // 错误信息
	TypeDone        ChunkType = "done"            // 对话结束信号
)

// Chunk 输出片段
type Chunk struct {
	Type    ChunkType      `json:"type"`
	Content string         `json:"content"`
	Meta    map[string]any `json:"meta,omitempty"`
}

// Writer 输出接口
type Writer interface {
	Write(Chunk)
}

// MultiWriter 多 Writer 组合
type MultiWriter struct {
	Writers []Writer
}

func NewMultiWriter(writers ...Writer) *MultiWriter {
	return &MultiWriter{Writers: writers}
}

func (m *MultiWriter) Write(c Chunk) {
	for _, w := range m.Writers {
		w.Write(c)
	}
}

// StdoutWriter 标准输出
type StdoutWriter struct{}

func NewStdoutWriter() Writer {
	return &StdoutWriter{}
}

func (s *StdoutWriter) Write(c Chunk) {
	fmt.Fprint(os.Stdout, c.Content)
}

// SSEWriter SSE 推送
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func NewSSEWriter(w http.ResponseWriter, flusher http.Flusher) Writer {
	return &SSEWriter{
		w:       w,
		flusher: flusher,
	}
}

func (s *SSEWriter) Write(c Chunk) {
	data, err := json.Marshal(c)
	if err != nil {
		data, _ = json.Marshal(Chunk{
			Type:    TypeError,
			Content: err.Error(),
		})
	}
	fmt.Fprintf(s.w, "data: %s\n\n", data)
	s.flusher.Flush()
}
