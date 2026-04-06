package store

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

type JSONLStore struct {
	mu    sync.Mutex
	Dir   string
	Cache map[string]*Session
}

// NewJSONLStore creates a new JSONLStore with the given directory path.
func NewJSONLStore(dir string) *JSONLStore {
	return &JSONLStore{
		Dir:   dir,
		Cache: make(map[string]*Session),
	}
}

// GetOrCreate returns the session for id, creating it if it does not exist.
func (s *JSONLStore) GetOrCreate(_ context.Context, sessionID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sess, ok := s.Cache[sessionID]; ok {
		return sess, nil
	}

	filePath := filepath.Join(s.Dir, sessionID+".jsonl")

	var sess *Session

	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		now := time.Now().UTC()
		header := map[string]interface{}{
			"type":       "session",
			"id":         sessionID,
			"created_at": now,
		}

		data, err := json.Marshal(header)
		if err != nil {
			return nil, err
		}

		if err = os.WriteFile(filePath, append(data, '\n'), 0o644); err != nil {
			return nil, err
		}

		sess = &Session{
			ID:        sessionID,
			CreatedAt: now,
			Messages:  make([]*schema.Message, 0),
		}
	} else {
		loaded, err := s.loadSession(filePath)
		if err != nil {
			return nil, err
		}
		sess = loaded
	}

	s.Cache[sessionID] = sess

	return sess, nil
}

// Append 追加一条或多条 message（支持批量写入）
func (s *JSONLStore) Append(ctx context.Context, sessionID string, messages ...*schema.Message) error {
	sess, err := s.GetOrCreate(ctx, sessionID)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(s.Dir, sessionID+".jsonl"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	// 使用 bufio.Writer 批量写入
	writer := bufio.NewWriter(f)
	for _, msg := range messages {
		sess.Messages = append(sess.Messages, msg)

		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal message: %w", err)
		}
		if _, err = writer.Write(data); err != nil {
			return err
		}
		if err = writer.WriteByte('\n'); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func (s *JSONLStore) loadSession(filePath string) (*Session, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty session file: %s", filePath)
	}

	var header struct {
		Type      string    `json:"type"`
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err = json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return nil, fmt.Errorf("bad session header in %s: %w", filePath, err)
	}

	sess := &Session{
		ID:        header.ID,
		CreatedAt: header.CreatedAt,
		Messages:  make([]*schema.Message, 0),
	}

	// 读取消息
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg schema.Message
		if err = json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("warn: failed to parse message in %s: %v", filePath, err)
			continue
		}
		sess.Messages = append(sess.Messages, &msg)
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return sess, nil
}
