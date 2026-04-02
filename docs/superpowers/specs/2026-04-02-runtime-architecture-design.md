---
name: Runtime Architecture Design
description: 解耦 CLI/SSE 交互层与 agent 服务层，通过 Sink 接口实现统一 Runtime 处理 eino 消息事件流
type: project
---

# Runtime Architecture Design

## Background

当前架构存在以下问题：

1. **Sink 接口只有单向 Emit** - 无法处理 SSE 的双向交互（如审批回调）
2. **Runner 职责过多** - 运行 agent + 事件处理 + 存储 + 审批，耦合度高
3. **SSESink 未实现** - 缺少 HTTP 层抽象
4. **没有 runtime 概念** - 缺少统一管理事件流生命周期的核心组件

## Goals

1. **统一 Runtime** - 作为服务层核心入口，管理完整会话生命周期
2. **解耦交互层** - CLI 和 SSE 通过统一的 Session 接口与 Runtime 交互
3. **扩展点清晰** - 支持未来 WebSocket 等新交互方式
4. **Sink 保持简单** - 单向输出接口，不承担审批逻辑

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Application Layer                        │
│                                                              │
│  ┌─────────────────┐              ┌─────────────────────┐    │
│  │    CLI Entry    │              │    HTTP Handler     │    │
│  │   (cli.go)      │              │   (api/handler.go)  │    │
│  └─────────────────┘              └─────────────────────┘    │
│           │                                 │                │
│           ▼                                 ▼                │
│  ┌─────────────────┐              ┌─────────────────────┐    │
│  │ CLISessionBuilder│             │ SSESessionBuilder   │    │
│  │  - OnInput 回调 │              │  - InputChan        │    │
│  │  - StdoutSink   │              │  - SubmitApproval() │    │
│  └─────────────────┘              │  - SSESink          │    │
│           │                        └─────────────────────┘    │
│           │                                 │                │
└───────────┼─────────────────────────────────┼────────────────┘
            │         ┌───────────────────────┤
            │         │                       │
            ▼         ▼                       ▼
┌─────────────────────────────────────────────────────────────┐
│                       Runtime Layer                          │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                       Runtime                          │  │
│  │  - Run(ctx, session, query)                           │  │
│  │  - Resume(ctx, session, checkpointID, resumeData)     │  │
│  │  - processEvents() → handleAgentEvent()               │  │
│  │  - handleInterrupt() → session.WaitInput()            │  │
│  │                                                        │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌───────────────┐  │  │
│  │  │   Store     │  │    Agent    │  │ CheckPointStore│  │  │
│  │  │   (注入)    │  │   (注入)    │  │    (注入)      │  │  │
│  │  └─────────────┘  └─────────────┘  └───────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                       Session                          │  │
│  │  - ID, Sink, InputChan, OnInput                       │  │
│  │  - WaitInput()                                        │  │
│  │  - Emit(), Collect(), Messages()                      │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│                       Infrastructure Layer                   │
│                                                              │
│  Sink: StdoutSink, SSESink, MultiSink                        │
│  Store: JSONLStore                                           │
│  Approval: ApprovalInfo, ApprovalResult                      │
│  Agent: agent.New()                                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### Session（双向交互容器）

Session 是一轮对话的交互容器，封装输出通道（Sink）和输入通道（InputChan/OnInput）。

```go
// runtime/session.go
type Session struct {
    ID        string
    Sink      sink.Sink
    InputChan chan InputEvent       // SSE 场景：channel 输入

    // CLI 场景：阻塞回调（可选）
    OnInput   func(ctx context.Context) (*InputEvent, error)

    ctx      context.Context
    messages []*schema.Message
}

type InputEvent struct {
    Type     InputType
    Data     any
}

type InputType string

const (
    InputApproval    InputType = "approval"
    InputUserMessage InputType = "user_message"
)

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

func (s *Session) Emit(c sink.Chunk) {
    s.Sink.Emit(c)
}

func (s *Session) Collect(msg *schema.Message) {
    s.messages = append(s.messages, msg)
}

func (s *Session) Messages() []*schema.Message {
    return s.messages
}

func (s *Session) Close() {
    close(s.InputChan)
}
```

**Why:** Session 封装双向交互，让交互差异（CLI 同步阻塞 vs SSE 异步 channel）透明化，Runtime 统一调用 WaitInput。

**How to apply:** 所有交互层通过 SessionBuilder 创建 Session，Runtime 只依赖 Session 接口。

---

### Runtime（服务层核心）

Runtime 是服务层唯一入口，管理完整会话生命周期。

```go
// runtime/runtime.go
type Runtime struct {
    agent      adk.Agent
    store      store.Store
    checkpointStore adk.CheckPointStore
}

type RuntimeOption func(*Runtime)

func WithStore(s store.Store) RuntimeOption
func WithCheckpointStore(cs adk.CheckPointStore) RuntimeOption

func NewRuntime(agent adk.Agent, opts ...RuntimeOption) (*Runtime, error)

// Run 执行一轮对话
func (r *Runtime) Run(ctx context.Context, session *Session, query string) error

// Resume 恢复中断的对话（审批后）
func (r *Runtime) Resume(ctx context.Context, session *Session, checkpointID string, resumeData map[string]any) error
```

**Run 内部流程：**

```go
func (r *Runtime) Run(ctx context.Context, session *Session, query string) error {
    // 1. 加载/创建会话历史
    sessHistory, _ := r.store.GetOrCreate(ctx, session.ID)
    r.store.Append(ctx, session.ID, schema.UserMessage(query))

    // 2. 运行 agent，获取事件流
    iter := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent: r.agent,
        EnableStreaming: true,
        CheckPointStore: r.checkpointStore,
    }).Run(ctx, sessHistory.Messages, adk.WithCheckPointID(session.ID))

    // 3. 处理事件流
    messages, interruptInfo, err := r.processEvents(ctx, session, iter)
    if err != nil {
        return err
    }

    // 4. 存储输出消息
    r.store.Append(ctx, session.ID, messages...)

    // 5. 处理中断（审批）
    if interruptInfo != nil {
        return r.handleInterrupt(ctx, session, interruptInfo)
    }

    return nil
}
```

**Why:** Runtime 统一管理会话生命周期，事件处理逻辑作为私有方法，不暴露。

**How to apply:** CLI 和 HTTP handler 都通过 Runtime.Run 执行对话，差异只在 Session 的创建方式。

---

### Sink（单向输出接口）

Sink 保持单向简单，只做输出。

```go
// sink/sink.go
type ChunkType string

const (
    TypeAssistant    ChunkType = "assistant"
    TypeToolCall     ChunkType = "tool_call"
    TypeToolResult   ChunkType = "tool_result"
    TypeMessage      ChunkType = "message"
    TypeApproval     ChunkType = "approval"      // 审批请求
    TypeApprovalRes  ChunkType = "approval_result" // 审批结果
    TypeError        ChunkType = "error"
    TypeDone         ChunkType = "done"          // 对话结束信号
)

type Chunk struct {
    Type    ChunkType
    Content string
    Meta    map[string]any  // 扩展字段：approval_id, tool_name 等
}

type Sink interface {
    Emit(Chunk)
}
```

**SSESink 实现：**

```go
// sink/sse.go
type SSESink struct {
    w       http.ResponseWriter
    flusher http.Flusher
}

func NewSSESink(w http.ResponseWriter, flusher http.Flusher) Sink

func (s *SSESink) Emit(c Chunk) {
    data, _ := json.Marshal(c)
    fmt.Fprintf(s.w, "data: %s\n\n", data)
    s.flusher.Flush()
}
```

**Why:** Sink 单向接口，职责单一。SSE 场景通过 SSE 格式推送 JSON，前端解析渲染不同类型消息。

**How to apply:** StdoutSink 直接输出，SSESink 推送 SSE 事件。未来 WebSocket 只需实现 WebSocketSink。

---

### SessionBuilder（交互层创建 Session）

SessionBuilder 让交互层决定如何创建 Session。

**CLI 实现：**

```go
// runtime/session_cli.go
type CLISessionBuilder struct {
    scanner *bufio.Scanner
}

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
            Data: &ApprovalResult{Approved: approved},
        }, nil
    }

    return session, nil
}
```

**SSE 实现：**

```go
// runtime/session_sse.go
type SSESessionBuilder struct {
    sessions sync.Map
}

func (b *SSESessionBuilder) Build(ctx context.Context, sessionID string) (*Session, error) {
    session := NewSession(ctx, sessionID, sink.NewSSESink())
    // 不注册 OnInput，使用 InputChan
    b.sessions.Store(sessionID, session)
    return session, nil
}

func (b *SSESessionBuilder) SubmitApproval(sessionID string, result *ApprovalResult) error {
    sess, ok := b.sessions.Load(sessionID)
    if !ok {
        return fmt.Errorf("session not found")
    }

    session := sess.(*Session)
    session.InputChan <- InputEvent{
        Type: InputApproval,
        Data: result,
    }
    return nil
}
```

**Why:** 交互差异封装在 SessionBuilder，Runtime 不感知 CLI/SSE。

**How to apply:** CLI 场景 OnInput 阻塞读 stdin，SSE 场景 HTTP handler 调用 SubmitApproval 写入 InputChan。

---

## File Structure

```
assistant/
├── cli.go                    # CLI 入口

├── runtime/
│   ├── runtime.go            # Runtime 结构 + Run/Resume 主逻辑
│   ├── event.go              # handleAgentEvent 等私有事件处理方法
│   ├── session.go            # Session 结构定义 + WaitInput
│   ├── session_cli.go        # CLISessionBuilder 实现
│   └── session_sse.go        # SSESessionBuilder 实现

├── sink/
│   ├── sink.go               # Sink 接口 + Chunk/ChunkType 定义 + MultiSink
│   ├── stdout.go             # StdoutSink 实现
│   └── sse.go                # SSESink 实现

├── store/
│   ├── store.go              # Store 接口 + Session 结构
│   └── jsonl.go              # JSONLStore 实现

├── approval/
│   └── approval.go           # ApprovalInfo + ApprovalResult 定义

├── agent/
│   └── agent.go              # agent.New() 创建逻辑

└── api/
    └── handler.go            # SSE handler + approval callback handler
```

**删除的文件：**

- `assistant/agent/runner.go` → 合并到 `runtime/runtime.go`
- `assistant/agent/runner_event.go` → 合并到 `runtime/event.go`
- `EventHandler` struct → 不需要，逻辑合并到 Runtime

---

## Data Flow

### CLI Flow

```
User Input (stdin)
       │
       ▼
  CLISessionBuilder.Build()
       │
       ▼
     Session (OnInput: 读 stdin)
       │
       ▼
     Runtime.Run()
       │
       ├──→ processEvents()
       │         │
       │         ├──→ session.Emit() → StdoutSink → stdout
       │         └──→ session.Collect()
       │
       ├──→ handleInterrupt()
       │         │
       │         ├──→ session.Emit(TypeApproval)
       │         └──→ session.WaitInput()
       │                   │
       │                   ▼
       │              OnInput() → 读 stdin → 返回 ApprovalResult
       │
       └──→ Resume() → 继续执行
       │
       └──→ store.Append()
```

### SSE Flow

```
HTTP /chat Request
       │
       ▼
  SSESessionBuilder.Build()
       │
       ▼
     Session (InputChan: channel)
       │
       ▼
     Runtime.Run() (goroutine)
       │
       ├──→ processEvents()
       │         │
       │         ├──→ session.Emit() → SSESink → SSE 推送
       │         └──→ session.Collect()
       │
       ├──→ handleInterrupt()
       │         │
       │         ├──→ session.Emit(TypeApproval) → SSE 推送审批请求
       │         └──→ session.WaitInput() → 阻塞等待 InputChan

HTTP /approval Request
       │
       ▼
  SubmitApproval()
       │
       ├──→ InputChan ←───┘
       │         │
       │         ▼
       │    WaitInput() 返回
       │         │
       ▼         ▼
     Runtime.Resume() → 继续执行
```

---

## Extension Points

1. **新交互方式** - 实现 SessionBuilder，决定 OnInput 或 InputChan
2. **新 Sink 实现** - 实现 Sink 接口，如 WebSocketSink
3. **新 Store 实现** - 实现 Store 接口，如 RedisStore
4. **事件处理扩展** - Runtime.handleAgentEvent 可扩展处理新事件类型

---

## Migration Plan

1. 创建 `runtime/` 目录，实现 Runtime 和 Session
2. 将 `agent/runner.go` 和 `agent/runner_event.go` 逻辑迁移到 `runtime/`
3. 实现 CLISessionBuilder 和 SSESessionBuilder
4. 实现 SSESink
5. 更新 `cli.go` 使用新架构
6. 创建 `api/handler.go` 实现 HTTP 层
7. 删除旧的 `agent/runner.go` 和 `agent/runner_event.go`

---

## Constraints

- Sink 实现必须是并发安全的（可能被多个 Run 并发调用）
- Session.InputChan 缓冲大小为 1，避免阻塞写入
- Runtime 不感知 CLI/SSE 差异，统一通过 Session.WaitInput 等待输入