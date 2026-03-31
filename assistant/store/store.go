package store

import "context"

type Store interface {
	Get(ctx context.Context, id string) (*Session, error)
	Set(ctx context.Context, s *Session) error
}
