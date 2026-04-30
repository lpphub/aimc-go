package store

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

type Store interface {
	Get(ctx context.Context, sessionID string) ([]*schema.Message, error)
	Append(ctx context.Context, sessionID string, messages ...*schema.Message) error
}
