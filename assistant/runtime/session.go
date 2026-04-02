package runtime

import (
	"aimc-go/assistant/sink"
	"context"

	"github.com/cloudwego/eino/schema"
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

// Session 双向交互容器
type Session struct {
	ID        string
	Sink      sink.Sink
	InputChan chan InputEvent // SSE 场景：channel 输入

	// CLI 场景：阻塞回调（可选）
	OnInput func(ctx context.Context) (*InputEvent, error)

	ctx      context.Context
	messages []*schema.Message
}

// NewSession 创建 Session
func NewSession(ctx context.Context, sessionID string, s sink.Sink) *Session {
	return &Session{
		ID:        sessionID,
		Sink:      s,
		InputChan: make(chan InputEvent, 1), // 缓冲 1，避免阻塞写入
		ctx:       ctx,
		messages:  make([]*schema.Message, 0, 20),
	}
}

// WaitInput 阻塞等待输入
func (s *Session) WaitInput(ctx context.Context) (*InputEvent, error) {
	// 如果注册了阻塞回调，直接调用（CLI 场景）
	if s.OnInput != nil {
		return s.OnInput(ctx)
	}

	// 否则阻塞等待 channel（SSE 场景）
	select {
	case input := <-s.InputChan:
		return &input, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Emit 输出 Chunk
func (s *Session) Emit(c sink.Chunk) {
	if s.Sink != nil {
		s.Sink.Emit(c)
	}
}

// Collect 收集消息
func (s *Session) Collect(msg *schema.Message) {
	s.messages = append(s.messages, msg)
}

// Messages 返回收集的消息
func (s *Session) Messages() []*schema.Message {
	return s.messages
}

// Close 关闭 session（关闭 InputChan）
func (s *Session) Close() {
	// 只关闭非 nil 的 channel（CLI 场景可能不使用）
	if s.InputChan != nil {
		close(s.InputChan)
	}
}