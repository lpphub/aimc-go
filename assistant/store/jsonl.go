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

// conversation 内部结构体（用于缓存和文件存储）
type conversation struct {
	ID        string
	CreatedAt time.Time
	Messages  []*schema.Message
}

type JSONLStore struct {
	Dir       string
	cache     *lru.Cache[string, *conversation]
	convLocks sync.Map // conversationID -> *sync.Mutex (细粒度锁)
}

// NewJSONLStore creates a new JSONLStore with the given directory path.
func NewJSONLStore(dir string) *JSONLStore {
	// 确保目录存在
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("warn: failed to create store dir: %v", err)
	}

	cache, _ := lru.New[string, *conversation](DefaultCacheSize)

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

// Get 获取历史消息，不存在返回空切片
func (s *JSONLStore) Get(_ context.Context, sessionID string) ([]*schema.Message, error) {
	// 1. 快速缓存检查（无锁读取）
	if conv, ok := s.cache.Get(sessionID); ok {
		return conv.Messages, nil
	}

	// 2. 获取该 conversation 的独立锁
	lock := s.getConvLock(sessionID)
	lock.Lock()
	defer lock.Unlock()

	// 3. 再次检查缓存（double-check）
	if conv, ok := s.cache.Get(sessionID); ok {
		return conv.Messages, nil
	}

	// 4. 检查文件是否存在
	filePath := filepath.Join(s.Dir, sessionID+".jsonl")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // 不存在返回空切片
	}

	// 5. 从文件加载
	conv, err := s.loadConversation(filePath)
	if err != nil {
		return nil, err
	}

	// 6. 加入缓存
	s.cache.Add(sessionID, conv)

	return conv.Messages, nil
}

// Append 追加消息，自动创建（如果不存在）
func (s *JSONLStore) Append(_ context.Context, sessionID string, messages ...*schema.Message) error {
	// 1. 获取该 conversation 的独立锁
	lock := s.getConvLock(sessionID)
	lock.Lock()
	defer lock.Unlock()

	// 2. 获取 conversation（从缓存或文件），但不加入缓存
	conv, cached := s.cache.Get(sessionID)
	if !cached {
		filePath := filepath.Join(s.Dir, sessionID+".jsonl")
		// 文件不存在则创建新的 conversation
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			now := time.Now().UTC()
			conv = &conversation{
				ID:        sessionID,
				CreatedAt: now,
				Messages:  make([]*schema.Message, 0),
			}
			// 写入 header
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
			// 文件存在则加载
			var err error
			conv, err = s.loadConversation(filePath)
			if err != nil {
				return fmt.Errorf("load conversation: %w", err)
			}
		}
	}

	// 3. 先写文件（全部写入并 flush 成功后，才更新缓存）
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

	// flush 落盘后才算写入成功
	if err = writer.Flush(); err != nil {
		return err
	}

	// 4. 文件写入成功，再更新缓存
	conv.Messages = append(conv.Messages, messages...)
	if !cached {
		s.cache.Add(sessionID, conv)
	}

	return nil
}

// loadConversation 从 JSONL 文件加载 conversation
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

	// 解析 header
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
