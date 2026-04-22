package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Writer 输出接口
type Writer interface {
	Write(Chunk) error
}

// MultiWriter 多 Writer 组合
type MultiWriter struct {
	Writers []Writer
}

func NewMultiWriter(writers ...Writer) *MultiWriter {
	return &MultiWriter{Writers: writers}
}

func (m *MultiWriter) Write(c Chunk) error {
	for _, w := range m.Writers {
		if err := w.Write(c); err != nil {
			return err
		}
	}
	return nil
}

// StdoutWriter 标准输出
type StdoutWriter struct{}

func NewStdoutWriter() Writer {
	return &StdoutWriter{}
}

func (s *StdoutWriter) Write(c Chunk) error {
	_, err := fmt.Print(c.Content)
	return err
}

// SSEWriter SSE 推送
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	ctx     context.Context
}

func NewSSEWriter(ctx context.Context, w http.ResponseWriter, flusher http.Flusher) Writer {
	return &SSEWriter{
		w:       w,
		flusher: flusher,
		ctx:     ctx,
	}
}

func (s *SSEWriter) Write(c Chunk) error {
	// context 取消就不再输出
	if s.ctx != nil && s.ctx.Err() != nil {
		return s.ctx.Err()
	}

	// 过滤 tool_call 和 tool_result 类型
	if c.Type == TypeToolCall || c.Type == TypeToolResult {
		return nil
	}

	data, err := json.Marshal(c)
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