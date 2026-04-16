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
	Cache map[string]*Conversation
}

// NewJSONLStore creates a new JSONLStore with the given directory path.
func NewJSONLStore(dir string) *JSONLStore {
	return &JSONLStore{
		Dir:   dir,
		Cache: make(map[string]*Conversation),
	}
}

// GetOrCreate returns the conversation for id, creating it if it does not exist.
func (s *JSONLStore) GetOrCreate(_ context.Context, conversationID string) (*Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conv, ok := s.Cache[conversationID]; ok {
		return conv, nil
	}

	filePath := filepath.Join(s.Dir, conversationID+".jsonl")

	var conv *Conversation

	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		now := time.Now().UTC()
		header := map[string]interface{}{
			"type":       "conversation",
			"id":         conversationID,
			"created_at": now,
		}

		data, err := json.Marshal(header)
		if err != nil {
			return nil, err
		}

		if err = os.WriteFile(filePath, append(data, '\n'), 0o644); err != nil {
			return nil, err
		}

		conv = &Conversation{
			ID:        conversationID,
			CreatedAt: now,
			Messages:  make([]*schema.Message, 0),
		}
	} else {
		loaded, err := s.loadConversation(filePath)
		if err != nil {
			return nil, err
		}
		conv = loaded
	}

	s.Cache[conversationID] = conv

	return conv, nil
}

// Append 追加一条或多条 message（支持批量写入）
func (s *JSONLStore) Append(ctx context.Context, conversationID string, messages ...*schema.Message) error {
	conv, err := s.GetOrCreate(ctx, conversationID)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(s.Dir, conversationID+".jsonl"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	// 使用 bufio.Writer 批量写入
	writer := bufio.NewWriter(f)
	for _, msg := range messages {
		conv.Messages = append(conv.Messages, msg)

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

func (s *JSONLStore) loadConversation(filePath string) (*Conversation, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty conversation file: %s", filePath)
	}

	var header struct {
		Type      string    `json:"type"`
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err = json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return nil, fmt.Errorf("bad conversation header in %s: %w", filePath, err)
	}

	conv := &Conversation{
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
		conv.Messages = append(conv.Messages, &msg)
	}

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	return conv, nil
}
