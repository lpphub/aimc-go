# CLI 命令功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 agent CLI 增加 `/new`、`/resume`、`/quit` 命令，支持会话管理。

**Architecture:** 新增 `command` 包，每个命令独立文件，cli.go 负责解析分发。扩展 store 支持会话列表查询。

**Tech Stack:** Go, eino ADK, JSONL 文件存储

---

## 文件结构

| 文件 | 职责 |
|------|------|
| `assistant/command/command.go` | 命令接口定义、注册、解析 |
| `assistant/command/new.go` | `/new` 命令实现 |
| `assistant/command/resume.go` | `/resume` 命令实现 |
| `assistant/command/quit.go` | `/quit` 命令实现 |
| `assistant/store/store.go` | 扩展 ListSessions、GetRecent |
| `assistant/cli.go` | 集成命令解析 |

---

### Task 1: Store 扩展 - ListSessions 和 GetRecent

**Files:**
- Modify: `assistant/store/store.go`
- Create: `assistant/store/store_test.go`

- [ ] **Step 1: Write the failing test for ListSessions**

```go
// assistant/store/store_test.go
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
	s1, _ := s.GetOrCreate(ctx, "session-1")
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd assistant/store && go test -v -run TestListSessions`
Expected: FAIL with "ListSessions not defined"

- [ ] **Step 3: Write minimal implementation**

```go
// assistant/store/store.go - 在文件末尾添加

// ListSessions 返回所有会话（按创建时间倒序）
func (s *JSONLStore) ListSessions() ([]*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}

	var sessions []*Session
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".jsonl") {
			continue
		}
		sessionID := strings.TrimSuffix(f.Name(), ".jsonl")
		filePath := filepath.Join(s.Dir, f.Name())

		sess, err := s.loadSession(filePath)
		if err != nil {
			continue
		}
		sessions = append(sessions, sess)
		s.Cache[sessionID] = sess
	}

	// 按创建时间倒序排序
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

// GetRecent 返回最近 N 个会话（按创建时间倒序）
func (s *JSONLStore) GetRecent(n int) ([]*Session, error) {
	sessions, err := s.ListSessions()
	if err != nil {
		return nil, err
	}

	if len(sessions) < n {
		return sessions, nil
	}

	return sessions[:n], nil
}
```

需要添加 import:
```go
import (
    // ... existing imports
    "sort"
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd assistant/store && go test -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add assistant/store/store.go assistant/store/store_test.go
git commit -m "feat(store): add ListSessions and GetRecent methods"
```

---

### Task 2: 命令包基础设施

**Files:**
- Create: `assistant/command/command.go`

- [ ] **Step 1: Write command interface and registry**

```go
// assistant/command/command.go
package command

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/store"
	"bufio"
	"context"
	"fmt"
	"strings"
)

// Command 命令函数签名
type Command func(ctx context.Context, deps *Dependencies, args string) error

// Dependencies 命令执行依赖
type Dependencies struct {
	Store     store.Store
	Runner    *agent.Runner
	SessionID *string // 指向当前 session ID
	Scanner   *bufio.Scanner
}

// Registry 命令注册表
var Registry = map[string]Command{
	"new":    NewCmd,
	"resume": ResumeCmd,
	"quit":   QuitCmd,
}

// Parse 解析输入，返回命令名和参数
func Parse(input string) (name string, args string) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return "", ""
	}

	input = strings.TrimPrefix(input, "/")
	parts := strings.SplitN(input, " ", 2)
	name = parts[0]
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}
	return name, args
}

// Execute 执行命令
func Execute(ctx context.Context, deps *Dependencies, name, args string) error {
	cmd, ok := Registry[name]
	if !ok {
		return fmt.Errorf("未知命令: /%s", name)
	}
	return cmd(ctx, deps, args)
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd assistant && go build ./command`
Expected: 编译失败（缺少 NewCmd, ResumeCmd, QuitCmd）

- [ ] **Step 3: Commit**

```bash
git add assistant/command/command.go
git commit -m "feat(command): add command interface and registry"
```

---

### Task 3: 实现 /new 命令

**Files:**
- Create: `assistant/command/new.go`

- [ ] **Step 1: Write /new implementation**

```go
// assistant/command/new.go
package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func NewCmd(ctx context.Context, deps *Dependencies, args string) error {
	newID := uuid.New().String()
	*deps.SessionID = newID
	fmt.Printf("✓ 已切换到新会话: %s\n", newID)
	return nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd assistant && go build ./command`
Expected: PASS（还有 ResumeCmd, QuitCmd 缺失）

- [ ] **Step 3: Commit**

```bash
git add assistant/command/new.go
git commit -m "feat(command): implement /new command"
```

---

### Task 4: 实现 /quit 命令

**Files:**
- Create: `assistant/command/quit.go`

- [ ] **Step 1: Write /quit implementation**

```go
// assistant/command/quit.go
package command

import (
	"context"
	"fmt"
	"os"
)

func QuitCmd(ctx context.Context, deps *Dependencies, args string) error {
	fmt.Println("再见！")
	os.Exit(0)
	return nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd assistant && go build ./command`
Expected: PASS（还有 ResumeCmd 缺失）

- [ ] **Step 3: Commit**

```bash
git add assistant/command/quit.go
git commit -m "feat(command): implement /quit command"
```

---

### Task 5: 实现 /resume 命令

**Files:**
- Create: `assistant/command/resume.go`

- [ ] **Step 1: Write /resume implementation**

```go
// assistant/command/resume.go
package command

import (
	"context"
	"fmt"
)

func ResumeCmd(ctx context.Context, deps *Dependencies, args string) error {
	if args == "" {
		// 无参数：恢复最近的会话（排除当前）
		sessions, err := deps.Store.ListSessions()
		if err != nil {
			return fmt.Errorf("获取会话列表失败: %w", err)
		}

		// 找到第一个非当前会话
		currentID := *deps.SessionID
		for _, s := range sessions {
			if s.ID != currentID {
				*deps.SessionID = s.ID
				fmt.Printf("✓ 已恢复会话: %s\n", s.ID)
				return nil
			}
		}

		return fmt.Errorf("没有其他可恢复的会话")
	}

	// 有参数：恢复指定会话
	sess, err := deps.Store.GetOrCreate(ctx, args)
	if err != nil {
		return fmt.Errorf("会话 %s 不存在: %w", args, err)
	}

	*deps.SessionID = sess.ID
	fmt.Printf("✓ 已恢复会话: %s\n", sess.ID)
	return nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd assistant && go build ./command`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add assistant/command/resume.go
git commit -m "feat(command): implement /resume command"
```

---

### Task 6: CLI 集成命令解析

**Files:**
- Modify: `assistant/cli.go`

- [ ] **Step 1: Update cli.go to use command package**

```go
// assistant/cli.go
package assistant

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/approval"
	"aimc-go/assistant/command"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

func Cli() {
	ctx := context.Background()

	assistantAgent, err := agent.New(ctx)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	sink := agent.StdoutSink()
	store := agent.JSONLStore("./data/sessions")
	approvalHandler := approval.NewCLIApprovalHandler(scanner, sink)

	runner, err := agent.NewRunner(assistantAgent,
		agent.WithStore(store),
		agent.WithSink(sink),
		agent.WithApprovalHandler(approvalHandler),
	)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	sessionID := "e69dfa6e-820a-4fcf-8a23-40107b0a324f"

	deps := &command.Dependencies{
		Store:     store,
		Runner:    runner,
		SessionID: &sessionID,
		Scanner:   scanner,
	}

	for {
		_, _ = fmt.Fprint(os.Stdout, "👤: ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 检查是否是命令
		name, args := command.Parse(line)
		if name != "" {
			if err := command.Execute(ctx, deps, name, args); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
			}
			continue
		}

		// 发送给 agent
		err = runner.Run(ctx, sessionID, line)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err = scanner.Err(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd assistant && go build .`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add assistant/cli.go
git commit -m "feat(cli): integrate command parsing"
```

---

### Task 7: 集成测试

**Files:**
- 无新增文件

- [ ] **Step 1: Run full build**

Run: `go build ./assistant`
Expected: PASS

- [ ] **Step 2: Run all tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: Manual smoke test**

启动 CLI，测试：
- 输入 `/new` - 应显示新 session ID
- 输入 `/resume` - 应恢复最近会话
- 输入 `/quit` - 应退出程序

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat: CLI commands complete"
```