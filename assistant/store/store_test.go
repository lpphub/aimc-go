package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestListSessions(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "store-test-"+uuid.New().String())
	_ = os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	s := &JSONLStore{Dir: dir, Cache: make(map[string]*Session)}

	// 创建两个会话
	ctx := context.Background()
	_, _ = s.GetOrCreate(ctx, "session-1")
	s2, _ := s.GetOrCreate(ctx, "session-2")

	// 等待确保时间差异
	time.Sleep(10 * time.Millisecond)

	list, err := s.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}

	// 按时间倒序，session-2 应该在前
	if list[0].ID != s2.ID {
		t.Errorf("expected first session to be %s, got %s", s2.ID, list[0].ID)
	}
}

func TestGetRecent(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "store-test-"+uuid.New().String())
	_ = os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	s := &JSONLStore{Dir: dir, Cache: make(map[string]*Session)}

	ctx := context.Background()
	s.GetOrCreate(ctx, "session-1")
	time.Sleep(10 * time.Millisecond)
	s.GetOrCreate(ctx, "session-2")
	time.Sleep(10 * time.Millisecond)
	s.GetOrCreate(ctx, "session-3")

	recent, err := s.GetRecent(2)
	if err != nil {
		t.Fatalf("GetRecent failed: %v", err)
	}

	if len(recent) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(recent))
	}

	// 最新的两个：session-3, session-2
	if recent[0].ID != "session-3" {
		t.Errorf("expected first to be session-3, got %s", recent[0].ID)
	}
}