package session

import (
	"context"
)

type Session struct {
	ID        string
	Transport Transport
}

func New(sessionID string, transport Transport) *Session {
	return &Session{
		ID:        sessionID,
		Transport: transport,
	}
}

func (s *Session) Emit(event Event) error {
	if s.Transport != nil {
		return s.Transport.Emit(event)
	}
	return nil
}

func (s *Session) WaitInput(ctx context.Context) (InputEvent, error) {
	return s.Transport.WaitInput(ctx)
}

func (s *Session) Close() {
	if s.Transport != nil {
		s.Transport.Close()
	}
}
