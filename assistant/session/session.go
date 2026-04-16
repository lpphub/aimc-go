package session

import (
	"context"
	"sync"
)

// InputType 输入事件类型
type InputType string

const (
	InputApproval    InputType = "approval"
	InputUserMessage InputType = "user_message"
)

// InputEvent 输入事件
type InputEvent struct {
	Type InputType
	Data any
}

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
