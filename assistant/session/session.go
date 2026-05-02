package session

import (
	"context"
)

type Session struct {
	ID       string
	Endpoint Endpoint
}

func New(sessionID string, endpoint Endpoint) *Session {
	return &Session{
		ID:       sessionID,
		Endpoint: endpoint,
	}
}

func (s *Session) Emit(event Event) error {
	if s.Endpoint != nil {
		return s.Endpoint.Emit(event)
	}
	return nil
}

func (s *Session) WaitInput(ctx context.Context) (InputEvent, error) {
	return s.Endpoint.WaitInput(ctx)
}

func (s *Session) Close() {
	if s.Endpoint != nil {
		s.Endpoint.Close()
	}
}
