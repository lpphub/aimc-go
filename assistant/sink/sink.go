package sink

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

// ChunkKind 输出片段类型
type ChunkKind string

const (
	KindAssistant  ChunkKind = "assistant"
	KindToolCall   ChunkKind = "tool_call"
	KindToolResult ChunkKind = "tool_result"
	KindMessage    ChunkKind = "message"
)

// Chunk 输出片段
type Chunk struct {
	Kind    ChunkKind
	Content string
}

// Sink 实时输出展示接口
// 职责：用户交互体验（stdout/SSE/WebSocket）
// 注意：实现必须是并发安全的，因为可能被多个 Run 并发调用
type Sink interface {
	Emit(c Chunk)
}

// MultiSink 多 sink 组合
type MultiSink struct {
	Sinks []Sink
}

func NewMultiSink(sinks ...Sink) *MultiSink {
	return &MultiSink{Sinks: sinks}
}

func (m *MultiSink) Emit(c Chunk) {
	for _, s := range m.Sinks {
		s.Emit(c)
	}
}

// StdoutSink 标准输出（带缓冲，高频场景更高效）
type StdoutSink struct {
	w  *bufio.Writer
	mu sync.Mutex
}

func NewStdoutSink() Sink {
	return &StdoutSink{
		w: bufio.NewWriterSize(os.Stdout, 4096),
	}
}

func (s *StdoutSink) Emit(c Chunk) {
	s.mu.Lock()
	fmt.Fprint(s.w, c.Content)
	s.w.Flush()
	s.mu.Unlock()
}

// SSESink SSE 推送（需配合 HTTP handler 使用）
// 使用方式：
//
//	sink := NewSSESink()
//	// 在 Runner 中：sink.Emit(chunk) 推送 SSE 事件
//	// 在 HTTP handler 中：flush SSE 连接
type SSESink struct {
	// TODO: 添加 SSE writer 和 flusher
}

func NewSSESink() Sink {
	return &SSESink{}
}

func (s *SSESink) Emit(c Chunk) {
	// TODO: 实现 SSE 推送
	// 需要持有 http.ResponseWriter 和 http.Flusher
	// fmt.Fprintf(w, "data: %s\n\n", c.Content)
	// f.Flush()
}
