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

// DefaultCacheSize 默认缓存容量
const DefaultCacheSize = 128

type JSONLStore struct {
	Dir       string
	cache     *lru.Cache[string, *Conversation]
	convLocks sync.Map // conversationID -> *sync.Mutex (细粒度锁)
}

// NewJSONLStore creates a new JSONLStore with the given directory path.
func NewJSONLStore(dir string) *JSONLStore {
	// 确保目录存在
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("warn: failed to create store dir: %v", err)
	}

	cache, _ := lru.New[string, *Conversation](DefaultCacheSize)

	return &JSONLStore{
		Dir:   dir,
		cache: cache,
	}
}

// getConvLock 获取指定 conversation 的独立锁
func (s *JSONLStore) getConvLock(id string) *sync.Mutex {
	lock, _ := s.convLocks.LoadOrStore(id, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// GetOrCreate 返回 conversation，不存在则创建
func (s *JSONLStore) GetOrCreate(_ context.Context, conversationID string) (*Conversation, error) {
	// 1. 快速缓存检查（无锁读取）
	if conv, ok := s.cache.Get(conversationID); ok {
		return conv, nil
	}

	// 2. 获取该 conversation 的独立锁
	lock := s.getConvLock(conversationID)
	lock.Lock()
	defer lock.Unlock()

	// 3. 再次检查缓存（double-check，避免重复加载）
	if conv, ok := s.cache.Get(conversationID); ok {
		return conv, nil
	}

	// 4. 检查文件
	filePath := filepath.Join(s.Dir, conversationID+".jsonl")

	var conv *Conversation

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 不存在 → 创建新 conversation
		now := time.Now().UTC()
		conv = &Conversation{
			ID:        conversationID,
			CreatedAt: now,
			Messages:  make([]*schema.Message, 0),
		}

		// 写入 header
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
	} else {
		// 存在 → 从文件加载
		conv, err = s.loadConversation(filePath)
		if err != nil {
			return nil, err
		}
	}

	// 5. 加入缓存
	s.cache.Add(conversationID, conv)

	return conv, nil
}

// Append 追加一条或多条 message（支持批量写入）
func (s *JSONLStore) Append(_ context.Context, conversationID string, messages ...*schema.Message) error {
	// 1. 获取该 conversation 的独立锁
	lock := s.getConvLock(conversationID)
	lock.Lock()
	defer lock.Unlock()

	// 2. 获取 conversation（从缓存或文件）
	conv, ok := s.cache.Get(conversationID)
	if !ok {
		filePath := filepath.Join(s.Dir, conversationID+".jsonl")
		var err error
		conv, err = s.loadConversation(filePath)
		if err != nil {
			return fmt.Errorf("load conversation: %w", err)
		}
		s.cache.Add(conversationID, conv)
	}

	// 3. 追加到文件
	filePath := filepath.Join(s.Dir, conversationID+".jsonl")
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	for _, msg := range messages {
		// 更新缓存
		conv.Messages = append(conv.Messages, msg)

		// 写入文件
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

// loadConversation 从 JSONL 文件加载 conversation
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

	// 解析 header
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

	// 解析消息
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
