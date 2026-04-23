package session

import (
	"context"
	"sync"
)

// Session 运行时 I/O 上下文（双向交互通道）
type Session struct {
	ID        string
	Sink      Sink                                           // agent 输出
	OnInput   func(ctx context.Context) (*InputEvent, error) // CLI 场景：阻塞回调
	InputChan chan InputEvent                                // SSE 场景：channel 输入
	closed    chan struct{}                                  // SSE 场景：关闭信号
	closeOnce sync.Once
}

// New 创建 Session
// withChan=true 使用 channel 输入（SSE/WebSocket），withChan=false 使用阻塞回调 OnInput（CLI）
func New(sessionID string, sink Sink, withChan bool) *Session {
	sess := &Session{
		ID:   sessionID,
		Sink: sink,
	}
	if withChan {
		sess.InputChan = make(chan InputEvent, 1)
		sess.closed = make(chan struct{})
	}
	return sess
}

// WaitInput 阻塞等待输入
func (s *Session) WaitInput(ctx context.Context) (*InputEvent, error) {
	if s.OnInput != nil {
		return s.OnInput(ctx)
	}

	select {
	case input := <-s.InputChan:
		return &input, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Emit 发送事件
func (s *Session) Emit(event Event) error {
	if s.Sink != nil {
		return s.Sink.Handle(event)
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
		if s.InputChan != nil {
			close(s.InputChan)
		}
	})
}
