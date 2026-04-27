package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Sink 事件接收接口
type Sink interface {
	Handle(Event) error
}

// MultiSink 多 Sink 组合
type MultiSink struct {
	Sinks []Sink
}

func NewMultiSink(sinks ...Sink) *MultiSink {
	return &MultiSink{Sinks: sinks}
}

func (m *MultiSink) Handle(e Event) error {
	for _, s := range m.Sinks {
		if err := s.Handle(e); err != nil {
			return err
		}
	}
	return nil
}

// StdoutSink 标准输出
type StdoutSink struct{}

func NewStdoutSink() Sink {
	return &StdoutSink{}
}

func (s *StdoutSink) Handle(e Event) error {
	switch e.Type {
	case TypeReasoning:
		_, err := fmt.Print("\033[90m" + e.Content + "\033[0m")
		return err
	default:
		_, err := fmt.Print(e.Content)
		return err
	}
}

// SSESink SSE 推送
type SSESink struct {
	w       http.ResponseWriter
	flusher http.Flusher
	ctx     context.Context
}

func NewSSESink(ctx context.Context, w http.ResponseWriter, flusher http.Flusher) Sink {
	return &SSESink{
		w:       w,
		flusher: flusher,
		ctx:     ctx,
	}
}

func (s *SSESink) Handle(e Event) error {
	// context 取消就不再输出
	if s.ctx != nil && s.ctx.Err() != nil {
		return s.ctx.Err()
	}

	// 过滤 tool_call 和 tool_result 类型
	if e.Type == TypeToolCall || e.Type == TypeToolResult {
		return nil
	}

	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}

	// flusher 存在才 flush
	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}
