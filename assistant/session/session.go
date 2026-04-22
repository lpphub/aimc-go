package session

import (
	"context"
	"sync"
)

// Session 运行时 I/O 上下文（双向交互通道）
type Session struct {
	ID        string
	Writer    Writer                                         // agent 输出
	Input     chan InputEvent                                // SSE 场景：channel 输入
	OnInput   func(ctx context.Context) (*InputEvent, error) // CLI 场景：阻塞回调
	closed    chan struct{}                                  // 关闭信号（SSE 场景）
	closeOnce sync.Once
}

// New 创建 Session（只初始化公共字段）
func New(sessionID string, writer Writer) *Session {
	return &Session{
		ID:     sessionID,
		Writer: writer,
	}
}

// NewSSE 创建 SSE 场景的 Session
func NewSSE(sessionID string, writer Writer) *Session {
	sess := New(sessionID, writer)
	sess.Input = make(chan InputEvent, 1)
	sess.closed = make(chan struct{})
	return sess
}

// WaitInput 阻塞等待输入
func (s *Session) WaitInput(ctx context.Context) (*InputEvent, error) {
	if s.OnInput != nil {
		return s.OnInput(ctx)
	}

	select {
	case input := <-s.Input:
		return &input, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Write 输出 Chunk
func (s *Session) Write(chunk Chunk) error {
	if s.Writer != nil {
		return s.Writer.Write(chunk)
	}
	return nil
}

// Closed 返回关闭信号 channel
func (s *Session) Closed() <-chan struct{} {
	return s.closed
}

// Close 关闭会话
func (s *Session) Close() {
	s.closeOnce.Do(func() {
		if s.closed != nil {
			close(s.closed)
		}
		if s.Input != nil {
			close(s.Input)
		}
	})
}