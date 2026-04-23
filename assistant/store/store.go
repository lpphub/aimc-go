package store

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// Store 消息存储接口
type Store interface {
	// Get 获取历史消息
	Get(ctx context.Context, sessionID string) ([]*schema.Message, error)
	// Append 追加消息，自动创建（如果不存在）
	Append(ctx context.Context, sessionID string, messages ...*schema.Message) error
}
