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
	lru "github.com/hashicorp/golang-lru/v2"
)

const DefaultCacheSize = 128

type conversation struct {
	ID        string
	CreatedAt time.Time
	Messages  []*schema.Message
}

type JSONLStore struct {
	Dir       string
	cache     *lru.Cache[string, *conversation]
	convLocks sync.Map // conversationID -> *sync.Mutex
}

func NewJSONLStore(dir string) *JSONLStore {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("warn: failed to create store dir: %v", err)
	}

	cache, _ := lru.New[string, *conversation](DefaultCacheSize)

	return &JSONLStore{
		Dir:   dir,
		cache: cache,
	}
}

func (s *JSONLStore) getConvLock(id string) *sync.Mutex {
	lock, _ := s.convLocks.LoadOrStore(id, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func (s *JSONLStore) Get(_ context.Context, sessionID string) ([]*schema.Message, error) {
	if conv, ok := s.cache.Get(sessionID); ok {
		return conv.Messages, nil
	}

	lock := s.getConvLock(sessionID)
	lock.Lock()
	defer lock.Unlock()

	filePath := filepath.Join(s.Dir, sessionID+".jsonl")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil
	}

	conv, err := s.loadConversation(filePath)
	if err != nil {
		return nil, err
	}

	s.cache.Add(sessionID, conv)

	return conv.Messages, nil
}

func (s *JSONLStore) Append(_ context.Context, sessionID string, messages ...*schema.Message) error {
	lock := s.getConvLock(sessionID)
	lock.Lock()
	defer lock.Unlock()

	conv, cached := s.cache.Get(sessionID)
	if !cached {
		filePath := filepath.Join(s.Dir, sessionID+".jsonl")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			now := time.Now().UTC()
			conv = &conversation{
				ID:        sessionID,
				CreatedAt: now,
				Messages:  make([]*schema.Message, 0),
			}
			header := map[string]interface{}{
				"type":       "conversation",
				"id":         sessionID,
				"created_at": now,
			}
			data, err := json.Marshal(header)
			if err != nil {
				return err
			}
			if err = os.WriteFile(filePath, append(data, '\n'), 0o644); err != nil {
				return err
			}
		} else {
			var err error
			conv, err = s.loadConversation(filePath)
			if err != nil {
				return fmt.Errorf("load conversation: %w", err)
			}
		}
	}

	filePath := filepath.Join(s.Dir, sessionID+".jsonl")
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	for _, msg := range messages {
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

	if err = writer.Flush(); err != nil {
		return err
	}

	conv.Messages = append(conv.Messages, messages...)
	if !cached {
		s.cache.Add(sessionID, conv)
	}

	return nil
}

func (s *JSONLStore) loadConversation(filePath string) (*conversation, error) {
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

	conv := &conversation{
		ID:        header.ID,
		CreatedAt: header.CreatedAt,
		Messages:  make([]*schema.Message, 0),
	}

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
