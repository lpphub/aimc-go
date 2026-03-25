package agent

import (
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

// SessionMeta provides summary info for the session list.
type SessionMeta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// Session holds the in-memory state for a single conversation.
type Session struct {
	SessionMeta
	messages []*schema.Message
	mu       sync.Mutex

	persist func(sessionID string, msg *schema.Message) error
}

// Append adds a message to memory and persists it to storage.
func (s *Session) Append(msg *schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, msg)

	if s.Title == "" {
		s.Title = s.genTitle()
	}

	if s.persist != nil {
		return s.persist(s.ID, msg)
	}
	return nil
}

// GetMessages returns a snapshot of all messages.
func (s *Session) GetMessages() []*schema.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]*schema.Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// genTitle derives a display title from the first user message.
func (s *Session) genTitle() string {
	for _, msg := range s.messages {
		if msg.Role == schema.User && msg.Content != "" {
			title := msg.Content
			if len([]rune(title)) > 60 {
				title = string([]rune(title)[:60]) + "..."
			}
			return title
		}
	}
	return "New Session"
}
