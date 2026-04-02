package store

import (
	"context"
	"time"

	"github.com/cloudwego/eino/schema"
)

type Session struct {
	ID        string
	CreatedAt time.Time
	Messages  []*schema.Message
}

type Store interface {
	//GetOrCreate 获取 session，不存在则创建
	GetOrCreate(ctx context.Context, sessionID string) (*Session, error)
	//Append 追加一条或多条 message
	Append(ctx context.Context, sessionID string, messages ...*schema.Message) error
}
