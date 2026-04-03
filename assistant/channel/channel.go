package channel

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

// Channel 双向交互通道
type Channel struct {
	ID      string
	Sink    Sink                                           // 输出
	Input   chan InputEvent                                // 输入 SSE：channel
	OnInput func(ctx context.Context) (*InputEvent, error) // 输入 CLI：阻塞回调

	closeOnce sync.Once
}

// NewChannel 创建 Channel
func NewChannel(sessionID string, s Sink) *Channel {
	return &Channel{
		ID:    sessionID,
		Sink:  s,
		Input: make(chan InputEvent, 1),
	}
}

// WaitInput 阻塞等待输入
func (c *Channel) WaitInput(ctx context.Context) (*InputEvent, error) {
	if c.OnInput != nil {
		return c.OnInput(ctx)
	}

	select {
	case input := <-c.Input:
		return &input, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Emit 输出 Chunk
func (c *Channel) Emit(chunk Chunk) {
	if c.Sink != nil {
		c.Sink.Emit(chunk)
	}
}

// Close 关闭通道
func (c *Channel) Close() {
	c.closeOnce.Do(func() {
		if c.Input != nil {
			close(c.Input)
		}
	})
}
