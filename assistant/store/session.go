package store

import (
	"time"

	"github.com/cloudwego/eino/schema"
)

type Session struct {
	ID        string
	CreatedAt time.Time
	messages  []*schema.Message
}
