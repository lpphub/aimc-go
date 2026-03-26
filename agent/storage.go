package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

type Store interface {
	GetOrCreate(id string) (*Session, error)
	List() ([]SessionMeta, error)
	Delete(id string) error
}

// JSONLStore manages persisted sessions backed by JSONL files.
type JSONLStore struct {
	dir   string
	mu    sync.Mutex
	cache map[string]*Session
}

// NewJSONLStore creates a new Store backed by the given directory (created if absent).
func NewJSONLStore(dir string) (Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create session dir: %w", err)
	}
	return &JSONLStore{
		dir:   dir,
		cache: make(map[string]*Session),
	}, nil
}

// GetOrCreate returns the session for id, creating it if it does not exist.
func (s *JSONLStore) GetOrCreate(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sess, ok := s.cache[id]; ok {
		return sess, nil
	}

	filePath := filepath.Join(s.dir, id+".jsonl")

	var sess *Session

	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		now := time.Now().UTC()
		header := map[string]interface{}{
			"type":       "session",
			"id":         id,
			"created_at": now,
		}

		data, err := json.Marshal(header)
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(filePath, append(data, '\n'), 0o644); err != nil {
			return nil, err
		}

		sess = &Session{
			SessionMeta: SessionMeta{ID: id, CreatedAt: now},
			messages:    make([]*schema.Message, 0),
			persist:     s.persistMessage,
		}
	} else {
		loaded, err := s.loadSession(filePath)
		if err != nil {
			return nil, err
		}
		loaded.persist = s.persistMessage
		sess = loaded
	}

	s.cache[id] = sess
	return sess, nil
}

// List returns metadata for all known sessions.
func (s *JSONLStore) List() ([]SessionMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	var metas []SessionMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}

		id := strings.TrimSuffix(e.Name(), ".jsonl")

		if sess, ok := s.cache[id]; ok {
			metas = append(metas, sess.SessionMeta)
			continue
		}

		loaded, err := s.loadSession(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		loaded.persist = s.persistMessage
		s.cache[id] = loaded

		metas = append(metas, loaded.SessionMeta)
	}

	return metas, nil
}

// Delete removes the session file and evicts it from the cache.
func (s *JSONLStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.dir, id+".jsonl")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	delete(s.cache, id)
	return nil
}

func (s *JSONLStore) persistMessage(sessionID string, msg *schema.Message) error {
	f, err := os.OpenFile(filepath.Join(s.dir, sessionID+".jsonl"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, _ := json.Marshal(msg)
	_, err = f.Write(append(data, '\n'))
	return err
}

func (s *JSONLStore) loadSession(filePath string) (*Session, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// 读取 header
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty session file: %s", filePath)
	}
	var header struct {
		Type      string    `json:"type"`
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return nil, fmt.Errorf("bad session header in %s: %w", filePath, err)
	}

	sess := &Session{
		SessionMeta: SessionMeta{ID: header.ID, CreatedAt: header.CreatedAt},
		messages:    make([]*schema.Message, 0),
		persist:     s.persistMessage,
	}

	// 读取消息
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg schema.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		sess.messages = append(sess.messages, &msg)
	}

	if sess.Title == "" {
		sess.Title = sess.genTitle()
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return sess, nil
}
