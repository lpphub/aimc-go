# Runtime Architecture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor agent service layer to decouple CLI/SSE interaction through unified Runtime + Session architecture.

**Architecture:** Session as bidirectional interaction container (Sink + InputChan/OnInput), Runtime as service core managing event stream lifecycle, SessionBuilder pattern for interaction layer abstraction.

**Tech Stack:** Go, eino (cloudwego agent SDK), SSE (Server-Sent Events), JSONL storage

---

## File Structure

| File | Responsibility | Action |
|------|---------------|--------|
| `runtime/session.go` | Session struct + WaitInput + Emit/Collect/Messages | Create |
| `runtime/runtime.go` | Runtime struct + Run/Resume + processEvents | Create |
| `runtime/event.go` | handleAgentEvent + handleAction + handleStreaming + handleNonStreaming | Create |
| `runtime/session_cli.go` | CLISessionBuilder (OnInput callback) | Create |
| `runtime/session_sse.go` | SSESessionBuilder (InputChan + SubmitApproval) | Create |
| `sink/sink.go` | Add new ChunkTypes | Modify |
| `sink/sse.go` | Full SSESink implementation with http.ResponseWriter | Modify |
| `cli.go` | Use new runtime architecture | Modify |
| `api/handler.go` | SSE + approval HTTP handlers | Create |
| `agent/runner.go` | Old Runner, logic moved to runtime | Delete |
| `agent/runner_event.go` | Old EventHandler, logic moved to runtime | Delete |

---

## Task 1: Update sink.go with new ChunkTypes

**Files:**
- Modify: `assistant/sink/sink.go`

- [ ] **Step 1: Add new ChunkType constants**

```go
const (
    TypeAssistant    ChunkType = "assistant"
    TypeToolCall     ChunkType = "tool_call"
    TypeToolResult   ChunkType = "tool_result"
    TypeMessage      ChunkType = "message"
    TypeApproval     ChunkType = "approval"        // 审批请求
    TypeApprovalRes  ChunkType = "approval_result" // 审批结果
    TypeError        ChunkType = "error"           // 错误信息
    TypeDone         ChunkType = "done"            // 对话结束信号
)
```

- [ ] **Step 2: Verify existing code compiles**

Run: `go build ./assistant/sink/...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add assistant/sink/sink.go
git commit -m "feat(sink): add new ChunkTypes for approval and error handling"
```

---

## Task 2: Implement SSESink

**Files:**
- Modify: `assistant/sink/sse.go`

- [ ] **Step 1: Implement SSESink with http.ResponseWriter**

```go
package sink

import (
    "encoding/json"
    "fmt"
    "net/http"
)

// SSESink SSE 推送 sink
type SSESink struct {
    w       http.ResponseWriter
    flusher http.Flusher
}

// NewSSESink 创建 SSE sink
func NewSSESink(w http.ResponseWriter, flusher http.Flusher) Sink {
    return &SSESink{
        w:       w,
        flusher: flusher,
    }
}

func (s *SSESink) Emit(c Chunk) {
    data, err := json.Marshal(c)
    if err != nil {
        // marshal error, emit as error chunk
        data, _ = json.Marshal(Chunk{
            Type:    TypeError,
            Content: err.Error(),
        })
    }
    fmt.Fprintf(s.w, "data: %s\n\n", data)
    s.flusher.Flush()
}
```

- [ ] **Step 2: Verify code compiles**

Run: `go build ./assistant/sink/...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add assistant/sink/sse.go
git commit -m "feat(sink): implement SSESink with SSE event format"
```

---

## Task 3: Create runtime directory and session.go

**Files:**
- Create: `assistant/runtime/session.go`

- [ ] **Step 1: Create runtime directory**

```bash
mkdir -p assistant/runtime
```

- [ ] **Step 2: Write session.go**

```go
package runtime

import (
    "aimc-go/assistant/sink"
    "context"

    "github.com/cloudwego/eino/schema"
)

// InputType 输入事件类型
type InputType string

const (
    InputApproval    InputType = "approval"
    InputUserMessage InputType = "user_message"
)

// InputEvent 输入事件
type InputEvent struct {
    Type InputType
    Data any
}

// Session 双向交互容器
type Session struct {
    ID        string
    Sink      sink.Sink
    InputChan chan InputEvent // SSE 场景：channel 输入

    // CLI 场景：阻塞回调（可选）
    OnInput func(ctx context.Context) (*InputEvent, error)

    ctx      context.Context
    messages []*schema.Message
}

// NewSession 创建 Session
func NewSession(ctx context.Context, sessionID string, s sink.Sink) *Session {
    return &Session{
        ID:        sessionID,
        Sink:      s,
        InputChan: make(chan InputEvent, 1), // 缓冲 1，避免阻塞写入
        ctx:       ctx,
        messages:  make([]*schema.Message, 0, 20),
    }
}

// WaitInput 阻塞等待输入
func (s *Session) WaitInput(ctx context.Context) (*InputEvent, error) {
    // 如果注册了阻塞回调，直接调用（CLI 场景）
    if s.OnInput != nil {
        return s.OnInput(ctx)
    }

    // 否则阻塞等待 channel（SSE 场景）
    select {
    case input := <-s.InputChan:
        return &input, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

// Emit 输出 Chunk
func (s *Session) Emit(c sink.Chunk) {
    if s.Sink != nil {
        s.Sink.Emit(c)
    }
}

// Collect 收集消息
func (s *Session) Collect(msg *schema.Message) {
    s.messages = append(s.messages, msg)
}

// Messages 返回收集的消息
func (s *Session) Messages() []*schema.Message {
    return s.messages
}

// Close 关闭 session（关闭 InputChan）
func (s *Session) Close() {
    // 只关闭非 nil 的 channel（CLI 场景可能不使用）
    if s.InputChan != nil {
        close(s.InputChan)
    }
}
```

- [ ] **Step 3: Verify code compiles**

Run: `go build ./assistant/runtime/...`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add assistant/runtime/session.go
git commit -m "feat(runtime): add Session as bidirectional interaction container"
```

---

## Task 4: Create runtime/event.go (event handling methods)

**Files:**
- Create: `assistant/runtime/event.go`

- [ ] **Step 1: Write event.go with private event handling methods**

```go
package runtime

import (
    "aimc-go/assistant/sink"
    "context"
    "errors"
    "fmt"
    "io"
    "strings"
    "unicode/utf8"

    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/schema"
)

// handleAgentEvent 处理 agent 事件
func (r *Runtime) handleAgentEvent(session *Session, event *adk.AgentEvent) (*adk.InterruptInfo, error) {
    // 1. error
    if event.Err != nil {
        session.Emit(sink.Chunk{Type: sink.TypeMessage, Content: fmt.Sprintf("⚠️ %s\n", event.Err)})
        if errors.Is(event.Err, adk.ErrExceedMaxIterations) {
            return nil, nil
        }
        return nil, event.Err
    }

    // 2. action (interrupt/transfer/exit)
    if event.Action != nil {
        return r.handleAction(session, event.Action), nil
    }

    // 3. message
    if event.Output == nil || event.Output.MessageOutput == nil {
        return nil, nil
    }

    mv := event.Output.MessageOutput

    // tool result
    if mv.Role == schema.Tool {
        result, err := mv.GetMessage()
        if err != nil {
            return nil, fmt.Errorf("get tool_result error: %w", err)
        }
        session.Collect(result)
        session.Emit(sink.Chunk{
            Type:    sink.TypeToolResult,
            Content: fmt.Sprintf("✅ [tool result] -> %s: %s\n", mv.ToolName, r.truncate(result.Content, 200)),
        })
        return nil, nil
    }

    // assistant message
    if mv.Role != schema.Assistant && mv.Role != "" {
        return nil, nil
    }

    if mv.IsStreaming {
        return nil, r.handleStreaming(session, mv)
    }
    return nil, r.handleNonStreaming(session, mv)
}

// handleAction 处理 action 事件
func (r *Runtime) handleAction(session *Session, action *adk.AgentAction) *adk.InterruptInfo {
    if action.Interrupted != nil {
        session.Emit(sink.Chunk{Type: sink.TypeMessage, Content: "⏸️ interrupted\n"})
        return action.Interrupted
    }

    if action.TransferToAgent != nil {
        session.Emit(sink.Chunk{
            Type:    sink.TypeMessage,
            Content: fmt.Sprintf("➡️ transfer to %s\n", action.TransferToAgent.DestAgentName),
        })
        return nil
    }

    if action.Exit {
        session.Emit(sink.Chunk{Type: sink.TypeMessage, Content: "🏁 exit\n"})
    }

    return nil
}

// handleStreaming 处理流式消息
func (r *Runtime) handleStreaming(session *Session, mv *adk.MessageVariant) error {
    mv.MessageStream.SetAutomaticClose()

    var contentBuf strings.Builder
    var accumulatedToolCalls []schema.ToolCall

    for {
        frame, err := mv.MessageStream.Recv()
        if errors.Is(err, io.EOF) {
            break
        }
        if err != nil {
            return err
        }
        if frame == nil {
            continue
        }

        if frame.Content != "" {
            contentBuf.WriteString(frame.Content)
            session.Emit(sink.Chunk{Type: sink.TypeAssistant, Content: frame.Content})
        }

        if len(frame.ToolCalls) > 0 {
            accumulatedToolCalls = append(accumulatedToolCalls, frame.ToolCalls...)
        }
    }

    session.Emit(sink.Chunk{Type: sink.TypeMessage, Content: "\n"})

    for _, tc := range accumulatedToolCalls {
        session.Emit(sink.Chunk{
            Type:    sink.TypeToolCall,
            Content: fmt.Sprintf("🔧 [tool call] -> %s: %s\n", tc.Function.Name, r.truncate(tc.Function.Arguments, 200)),
        })
    }

    session.Collect(&schema.Message{
        Role:      schema.Assistant,
        Content:   contentBuf.String(),
        ToolCalls: accumulatedToolCalls,
    })

    return nil
}

// handleNonStreaming 处理非流式消息
func (r *Runtime) handleNonStreaming(session *Session, mv *adk.MessageVariant) error {
    if mv.Message == nil {
        return nil
    }

    session.Emit(sink.Chunk{Type: sink.TypeAssistant, Content: mv.Message.Content})

    for _, tc := range mv.Message.ToolCalls {
        session.Emit(sink.Chunk{
            Type:    sink.TypeToolCall,
            Content: fmt.Sprintf("\n🔧 [tool call] -> %s: %s\n", tc.Function.Name, r.truncate(tc.Function.Arguments, 200)),
        })
    }

    session.Collect(mv.Message)

    return nil
}

// truncate 截断字符串
func (r *Runtime) truncate(s string, maxLen int) string {
    if utf8.RuneCountInString(s) <= maxLen {
        return s
    }
    runes := []rune(s)
    if len(runes) > maxLen {
        return string(runes[:maxLen]) + "..."
    }
    return s
}

// processEvents 处理事件流
func (r *Runtime) processEvents(ctx context.Context, session *Session, iter *adk.AsyncIterator[*adk.AgentEvent]) (
    []*schema.Message, *adk.InterruptInfo, error,
) {
    session.Emit(sink.Chunk{Type: sink.TypeMessage, Content: "🤖: "})

    for {
        event, ok := iter.Next()
        if !ok {
            break
        }

        interruptInfo, err := r.handleAgentEvent(session, event)
        if err != nil {
            return nil, nil, err
        }
        if interruptInfo != nil {
            return session.Messages(), interruptInfo, nil
        }
    }

    return session.Messages(), nil, nil
}
```

- [ ] **Step 2: Verify code compiles**

Run: `go build ./assistant/runtime/...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add assistant/runtime/event.go
git commit -m "feat(runtime): add event handling methods from runner_event.go"
```

---

## Task 5: Create runtime/runtime.go (Runtime struct + Run/Resume)

**Files:**
- Create: `assistant/runtime/runtime.go`

- [ ] **Step 1: Write runtime.go**

```go
package runtime

import (
    "aimc-go/assistant/approval"
    "aimc-go/assistant/sink"
    "aimc-go/assistant/store"
    "context"
    "fmt"

    adkstore "github.com/cloudwego/eino-examples/adk/common/store"
    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/schema"
)

// Runtime 服务层核心
type Runtime struct {
    agent           adk.Agent
    store           store.Store
    checkpointStore adk.CheckPointStore
}

// RuntimeOption Runtime 配置选项
type RuntimeOption func(*Runtime)

// WithStore 设置存储
func WithStore(s store.Store) RuntimeOption {
    return func(r *Runtime) {
        r.store = s
    }
}

// WithCheckpointStore 设置 checkpoint 存储
func WithCheckpointStore(cs adk.CheckPointStore) RuntimeOption {
    return func(r *Runtime) {
        r.checkpointStore = cs
    }
}

// NewRuntime 创建 Runtime
func NewRuntime(agent adk.Agent, opts ...RuntimeOption) (*Runtime, error) {
    r := &Runtime{
        agent: agent,
        checkpointStore: adkstore.NewInMemoryStore(), // 默认内存 checkpoint
    }

    for _, opt := range opts {
        opt(r)
    }

    if r.store == nil {
        return nil, fmt.Errorf("store is required, use WithStore() to set")
    }

    return r, nil
}

// Run 执行一轮对话
func (r *Runtime) Run(ctx context.Context, session *Session, query string) error {
    // 1. 加载/创建会话历史
    sessHistory, err := r.store.GetOrCreate(ctx, session.ID)
    if err != nil {
        return fmt.Errorf("get or create session: %w", err)
    }

    err = r.store.Append(ctx, session.ID, schema.UserMessage(query))
    if err != nil {
        return fmt.Errorf("append user message: %w", err)
    }

    // 2. 运行 agent，获取事件流
    innerRunner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent:           r.agent,
        EnableStreaming: true,
        CheckPointStore: r.checkpointStore,
    })

    iter := innerRunner.Run(ctx, sessHistory.Messages, adk.WithCheckPointID(session.ID))

    // 3. 处理事件流
    messages, interruptInfo, err := r.processEvents(ctx, session, iter)
    if err != nil {
        return err
    }

    // 4. 存储输出消息
    if len(messages) > 0 {
        err = r.store.Append(ctx, session.ID, messages...)
        if err != nil {
            return fmt.Errorf("append messages: %w", err)
        }
    }

    // 5. 处理中断（审批）
    if interruptInfo != nil {
        return r.handleInterrupt(ctx, session, interruptInfo)
    }

    // 6. 发送完成信号
    session.Emit(sink.Chunk{Type: sink.TypeDone})

    return nil
}

// handleInterrupt 处理中断（审批）
func (r *Runtime) handleInterrupt(ctx context.Context, session *Session, interruptInfo *adk.InterruptInfo) error {
    for _, ic := range interruptInfo.InterruptContexts {
        if !ic.IsRootCause {
            continue
        }

        // 发送审批请求
        approvalID := ic.ID
        info, ok := ic.Info.(*approval.ApprovalInfo)
        if !ok {
            return fmt.Errorf("unexpected interrupt info type: %T", ic.Info)
        }

        session.Emit(sink.Chunk{
            Type:    sink.TypeApproval,
            Content: info.String(),
            Meta:    map[string]any{"approval_id": approvalID, "tool_name": info.ToolName},
        })

        // 等待审批结果
        input, err := session.WaitInput(ctx)
        if err != nil {
            return fmt.Errorf("wait approval input: %w", err)
        }

        if input.Type != InputApproval {
            return fmt.Errorf("unexpected input type: %s", input.Type)
        }

        result, ok := input.Data.(*approval.ApprovalResult)
        if !ok {
            return fmt.Errorf("unexpected approval result type: %T", input.Data)
        }

        // 发送审批结果通知
        if result.Approved {
            session.Emit(sink.Chunk{Type: sink.TypeApprovalRes, Content: "✔️ Approved, executing...\n"})
        } else {
            session.Emit(sink.Chunk{Type: sink.TypeApprovalRes, Content: "✖️ Rejected\n"})
        }

        // 恢复执行
        messages, newInterrupt, err := r.Resume(ctx, session, session.ID, map[string]any{
            approvalID: result,
        })
        if err != nil {
            return fmt.Errorf("resume after approval: %w", err)
        }

        // 存储恢复后的消息
        if len(messages) > 0 {
            err = r.store.Append(ctx, session.ID, messages...)
            if err != nil {
                return fmt.Errorf("append resumed messages: %w", err)
            }
        }

        // 递归处理后续中断
        if newInterrupt != nil {
            return r.handleInterrupt(ctx, session, newInterrupt)
        }
    }

    // 发送完成信号
    session.Emit(sink.Chunk{Type: sink.TypeDone})

    return nil
}

// Resume 恢复中断的对话
func (r *Runtime) Resume(ctx context.Context, session *Session, checkpointID string, resumeData map[string]any) (
    []*schema.Message, *adk.InterruptInfo, error,
) {
    innerRunner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent:           r.agent,
        EnableStreaming: true,
        CheckPointStore: r.checkpointStore,
    })

    events, err := innerRunner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{
        Targets: resumeData,
    })
    if err != nil {
        return nil, nil, fmt.Errorf("resume with params: %w", err)
    }

    return r.processEvents(ctx, session, events)
}
```

- [ ] **Step 2: Verify code compiles**

Run: `go build ./assistant/runtime/...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add assistant/runtime/runtime.go
git commit -m "feat(runtime): add Runtime struct with Run/Resume methods"
```

---

## Task 6: Create runtime/session_cli.go (CLISessionBuilder)

**Files:**
- Create: `assistant/runtime/session_cli.go`

- [ ] **Step 1: Write session_cli.go**

```go
package runtime

import (
    "aimc-go/assistant/approval"
    "aimc-go/assistant/sink"
    "bufio"
    "context"
    "fmt"
    "strings"
)

// CLISessionBuilder CLI 场景的 SessionBuilder
type CLISessionBuilder struct {
    scanner *bufio.Scanner
}

// NewCLISessionBuilder 创建 CLI SessionBuilder
func NewCLISessionBuilder(scanner *bufio.Scanner) *CLISessionBuilder {
    return &CLISessionBuilder{scanner: scanner}
}

// Build 创建 CLI Session
func (b *CLISessionBuilder) Build(ctx context.Context, sessionID string) (*Session, error) {
    session := NewSession(ctx, sessionID, sink.NewStdoutSink())

    // 注册阻塞回调，直接读 stdin
    session.OnInput = func(ctx context.Context) (*InputEvent, error) {
        if !b.scanner.Scan() {
            return nil, fmt.Errorf("failed to read input")
        }
        response := strings.TrimSpace(b.scanner.Text())

        approved := response == "y" || response == "yes"
        return &InputEvent{
            Type: InputApproval,
            Data: &approval.ApprovalResult{Approved: approved},
        }, nil
    }

    return session, nil
}
```

- [ ] **Step 2: Verify code compiles**

Run: `go build ./assistant/runtime/...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add assistant/runtime/session_cli.go
git commit -m "feat(runtime): add CLISessionBuilder with OnInput callback"
```

---

## Task 7: Create runtime/session_sse.go (SSESessionBuilder)

**Files:**
- Create: `assistant/runtime/session_sse.go`

- [ ] **Step 1: Write session_sse.go**

```go
package runtime

import (
    "aimc-go/assistant/approval"
    "aimc-go/assistant/sink"
    "context"
    "fmt"
    "net/http"
    "sync"
)

// SSESessionBuilder SSE 场景的 SessionBuilder
type SSESessionBuilder struct {
    sessions sync.Map // sessionID -> *Session
}

// NewSSESessionBuilder 创建 SSE SessionBuilder
func NewSSESessionBuilder() *SSESessionBuilder {
    return &SSESessionBuilder{}
}

// Build 创建 SSE Session
func (b *SSESessionBuilder) Build(ctx context.Context, sessionID string, w http.ResponseWriter, flusher http.Flusher) (*Session, error) {
    session := NewSession(ctx, sessionID, sink.NewSSESink(w, flusher))
    // 不注册 OnInput，使用 InputChan

    b.sessions.Store(sessionID, session)
    return session, nil
}

// SubmitApproval 提交审批结果（供 HTTP handler 调用）
func (b *SSESessionBuilder) SubmitApproval(sessionID string, result *approval.ApprovalResult) error {
    sess, ok := b.sessions.Load(sessionID)
    if !ok {
        return fmt.Errorf("session not found: %s", sessionID)
    }

    session := sess.(*Session)
    session.InputChan <- InputEvent{
        Type: InputApproval,
        Data: result,
    }
    return nil
}

// GetSession 获取已存在的 session（用于审批回调）
func (b *SSESessionBuilder) GetSession(sessionID string) (*Session, error) {
    sess, ok := b.sessions.Load(sessionID)
    if !ok {
        return nil, fmt.Errorf("session not found: %s", sessionID)
    }
    return sess.(*Session), nil
}

// RemoveSession 移除 session（清理）
func (b *SSESessionBuilder) RemoveSession(sessionID string) {
    b.sessions.Delete(sessionID)
}
```

- [ ] **Step 2: Verify code compiles**

Run: `go build ./assistant/runtime/...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add assistant/runtime/session_sse.go
git commit -m "feat(runtime): add SSESessionBuilder with InputChan and SubmitApproval"
```

---

## Task 8: Update cli.go to use new runtime

**Files:**
- Modify: `assistant/cli.go`

- [ ] **Step 1: Rewrite cli.go with new architecture**

```go
package assistant

import (
    "aimc-go/assistant/agent"
    "aimc-go/assistant/runtime"
    "aimc-go/assistant/store"
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
    builder := runtime.NewCLISessionBuilder(scanner)
    jsonlStore := store.NewJSONLStore("./data/sessions")

    rt, err := runtime.NewRuntime(assistantAgent, runtime.WithStore(jsonlStore))
    if err != nil {
        fmt.Fprint(os.Stderr, err)
        os.Exit(1)
    }

    sessionID := "e69dfa6e-820a-4fcf-8a23-40107b0a324f"

    for {
        _, _ = fmt.Fprint(os.Stdout, "👤: ")
        if !scanner.Scan() {
            break
        }
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            break
        }

        session, err := builder.Build(ctx, sessionID)
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }

        err = rt.Run(ctx, session, line)
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }

        session.Close()
    }

    if err = scanner.Err(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

- [ ] **Step 2: Verify code compiles**

Run: `go build ./assistant/...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add assistant/cli.go
git commit -m "refactor(cli): use new runtime architecture with SessionBuilder"
```

---

## Task 9: Create api/handler.go (HTTP handlers)

**Files:**
- Create: `assistant/api/handler.go`

- [ ] **Step 1: Create api directory**

```bash
mkdir -p assistant/api
```

- [ ] **Step 2: Write handler.go**

```go
package api

import (
    "aimc-go/assistant/approval"
    "aimc-go/assistant/runtime"
    "aimc-go/assistant/sink"
    "context"
    "encoding/json"
    "net/http"
)

// Handler HTTP handler
type Handler struct {
    rt             *runtime.Runtime
    sessionBuilder *runtime.SSESessionBuilder
}

// NewHandler 创建 Handler
func NewHandler(rt *runtime.Runtime) *Handler {
    return &Handler{
        rt:             rt,
        sessionBuilder: runtime.NewSSESessionBuilder(),
    }
}

// ChatRequest 聊天请求
type ChatRequest struct {
    SessionID string `json:"session_id"`
    Query     string `json:"query"`
}

// Chat SSE 聊天接口
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
    // 设置 SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming unsupported", http.StatusInternalServerError)
        return
    }

    var req ChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    ctx := r.Context()

    session, err := h.sessionBuilder.Build(ctx, req.SessionID, w, flusher)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 异步运行 runtime
    go func() {
        err := h.rt.Run(ctx, session, req.Query)
        if err != nil {
            session.Emit(sink.Chunk{
                Type:    sink.TypeError,
                Content: err.Error(),
            })
        }
        // 运行结束后清理 session
        h.sessionBuilder.RemoveSession(req.SessionID)
        session.Close()
    }()

    // 阻塞保持连接
    <-ctx.Done()
}

// ApprovalRequest 宭批请求
type ApprovalRequest struct {
    SessionID  string `json:"session_id"`
    ApprovalID string `json:"approval_id"`
    Approved   bool   `json:"approved"`
    Reason     string `json:"reason,omitempty"`
}

// Approval 宭批回调接口
func (h *Handler) Approval(w http.ResponseWriter, r *http.Request) {
    var req ApprovalRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    result := &approval.ApprovalResult{
        Approved:         req.Approved,
        DisapproveReason: nil,
    }
    if req.Reason != "" {
        result.DisapproveReason = &req.Reason
    }

    err := h.sessionBuilder.SubmitApproval(req.SessionID, result)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

- [ ] **Step 3: Verify code compiles**

Run: `go build ./assistant/api/...`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add assistant/api/handler.go
git commit -m "feat(api): add SSE chat and approval HTTP handlers"
```

---

## Task 10: Delete old runner files

**Files:**
- Delete: `assistant/agent/runner.go`
- Delete: `assistant/agent/runner_event.go`

- [ ] **Step 1: Delete runner.go**

```bash
git rm assistant/agent/runner.go
```

- [ ] **Step 2: Delete runner_event.go**

```bash
git rm assistant/agent/runner_event.go
```

- [ ] **Step 3: Verify entire project compiles**

Run: `go build ./...`
Expected: Build succeeds

- [ ] **Step 4: Commit deletion**

```bash
git commit -m "refactor: remove old runner files, logic moved to runtime"
```

---

## Task 11: Final verification and cleanup

- [ ] **Step 1: Run full build**

Run: `go build ./...`
Expected: Build succeeds with no errors

- [ ] **Step 2: Run tests (if any exist)**

Run: `go test ./...`
Expected: All tests pass

- [ ] **Step 3: Verify CLI still works**

Run: `go run .` or appropriate CLI entry
Expected: CLI prompts for input and processes queries

- [ ] **Step 4: Final commit (if any remaining changes)**

```bash
git status
# If clean, no action needed
```

---

## Summary

This plan creates a clean separation:
- **Runtime Layer**: `runtime/runtime.go`, `runtime/event.go`, `runtime/session.go`
- **Interaction Layer**: `runtime/session_cli.go`, `runtime/session_sse.go`, `api/handler.go`
- **Infrastructure Layer**: `sink/`, `store/`, `approval/`, `agent/`

Key extension points remain:
- New interaction modes → implement SessionBuilder
- New output formats → implement Sink
- New storage backends → implement Store