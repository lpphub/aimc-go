# Assistant 架构重构：解耦交互层与 Agent 核心层

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 重构 assistant 包，将交互层（CLI/SSE）与 Agent 核心层解耦，使核心层只产生结构化事件，交互层决定如何展示。

**Architecture:** 采用事件驱动架构，Runner 产生结构化事件，通过回调函数传递给交互层。交互层负责格式化、展示、审批等 UI 相关逻辑。核心层不依赖任何展示相关的包。

**Tech Stack:** Go, Eino ADK, SSE

---

## 当前架构问题分析

### 耦合点

1. **cli.go → agent 包**
   - 直接调用 `agent.StdoutSink()`, `agent.JSONLStore()`
   - 工厂函数应该在各自的包中

2. **EventHandler → Sink**
   - `runner_event.go` 直接调用 `ec.Emit(sink.Chunk{...})`
   - 格式化逻辑（emoji、样式）耦合在核心层

3. **ApprovalHandler → Sink**
   - `approval.go` 直接导入 sink 包
   - 审批信息格式化在 approval 包中

4. **command.Dependencies → agent.Runner**
   - command 包直接依赖 agent.Runner

### 设计目标

- **核心层 (core/)**: 只产生结构化事件，不关心如何展示
- **交互层 (interface/)**: 订阅事件，决定如何展示
- **单向依赖**: 核心层 → Eino ADK；交互层 → 核心层

---

## 新架构设计

### 目录结构

```
assistant/
├── core/                          # 核心层：纯业务逻辑
│   ├── agent.go                   # Agent 构建（从 agent/agent.go 迁移）
│   ├── runner.go                  # Runner 运行器
│   ├── event.go                   # 事件类型定义
│   └── event_handler.go           # EventHandler 逻辑
├── interface/                     # 交互层
│   ├── cli/
│   │   └── cli.go                 # CLI 交互入口
│   ├── sse/
│   │   ├── handler.go             # SSE HTTP handler
│   │   └── sse_sink.go            # SSE Sink 实现
│   ├── approval.go                # 审批处理（CLI/SSE）
│   ├── formatter.go               # 格式化逻辑（emoji、样式）
│   └── sink.go                    # Sink 接口和 Stdout 实现
├── store/                         # 存储层（保持不变）
├── command/                       # 命令系统
├── agent/                         # Agent 子模块（保持）
│   ├── llm/
│   ├── middleware/
│   ├── prompts/
│   └── tools/
└── cli.go                         # 入口（调用 interface/cli）
```

### 事件类型定义

```go
// core/event.go
package core

import "time"

// Event 结构化事件，核心层产生，交互层消费
type Event struct {
    Type      EventType
    Data      EventData
    Timestamp time.Time
}

type EventType string

const (
    EventAgentOutput  EventType = "agent_output"   // 助手回复
    EventToolCall     EventType = "tool_call"      // 工具调用
    EventToolResult   EventType = "tool_result"    // 工具结果
    EventError        EventType = "error"          // 错误
    EventInterrupt    EventType = "interrupt"      // 中断（需要审批）
    EventAction       EventType = "action"         // 动作（转移、退出）
    EventNewline      EventType = "newline"        // 换行（流式结束标记）
)

// EventData 事件数据接口
type EventData interface {
    eventMarker()
}

// AgentOutputEvent 助手输出事件
type AgentOutputEvent struct {
    Content     string
    IsStreaming bool
}

// ToolCallEvent 工具调用事件
type ToolCallEvent struct {
    ToolName  string
    Arguments string
}

// ToolResultEvent 工具结果事件
type ToolResultEvent struct {
    ToolName string
    Result   string
    Success  bool
}

// InterruptEvent 中断事件（需要审批）
type InterruptEvent struct {
    InterruptID string
    ToolName    string
    Arguments   string
}

// ErrorEvent 错误事件
type ErrorEvent struct {
    Error   error
    Message string
}

// ActionEvent 动作事件
type ActionEvent struct {
    Action   string // "transfer", "exit", "interrupted"
    AgentName string // 转移目标 agent 名称
}

func (AgentOutputEvent) eventMarker()  {}
func (ToolCallEvent) eventMarker()     {}
func (ToolResultEvent) eventMarker()   {}
func (InterruptEvent) eventMarker()    {}
func (ErrorEvent) eventMarker()        {}
func (ActionEvent) eventMarker()       {}
```

### EventHandler 设计

```go
// core/event_handler.go
package core

import "github.com/cloudwego/eino/adk"

// EventHandler 事件处理回调函数
type EventHandler func(event Event)

// AgentEventHandler Eino Agent 事件处理器
type AgentEventHandler struct {
    handler EventHandler
}

func NewAgentEventHandler(handler EventHandler) *AgentEventHandler {
    return &AgentEventHandler{handler: handler}
}

func (h *AgentEventHandler) HandleEvent(event *adk.AgentEvent, sink func(Event)) {
    // 将 Eino AgentEvent 转换为结构化 Event
    // 调用 sink 发送事件
}
```

### Runner 设计

```go
// core/runner.go
package core

import (
    "aimc-go/assistant/store"
    "context"
    "github.com/cloudwego/eino/adk"
)

type Runner struct {
    inner   *adk.Runner
    store   store.Store
    handler EventHandler
}

type RunnerOption func(*Runner)

func WithStore(s store.Store) RunnerOption {
    return func(r *Runner) {
        r.store = s
    }
}

func NewRunner(agent adk.Agent, handler EventHandler, opts ...RunnerOption) (*Runner, error) {
    r := &Runner{
        inner:   adk.NewRunner(...),
        handler: handler,
    }
    
    for _, opt := range opts {
        opt(r)
    }
    
    if r.store == nil {
        return nil, fmt.Errorf("store is required")
    }
    
    return r, nil
}

func (r *Runner) Run(ctx context.Context, sessionID, query string) error {
    // 1. 获取/创建会话
    // 2. 添加用户消息
    // 3. 运行 Agent
    // 4. 将 Eino 事件转换为结构化 Event
    // 5. 调用 r.handler(event) 发送事件
    // 6. 存储消息
}

func (r *Runner) Resume(ctx context.Context, sessionID string, approved bool) error {
    // 恢复中断的对话
}
```

### Sink 接口（移动到 interface 包）

```go
// interface/sink.go
package iface

// ChunkKind 输出片段类型
type ChunkKind string

const (
    KindAssistant  ChunkKind = "assistant"
    KindToolCall   ChunkKind = "tool_call"
    KindToolResult ChunkKind = "tool_result"
    KindMessage    ChunkKind = "message"
)

// Chunk 输出片段
type Chunk struct {
    Kind    ChunkKind
    Content string
}

// Sink 输出接口
type Sink interface {
    Emit(c Chunk)
}

// StdoutSink 标准输出
type StdoutSink struct { ... }

// MultiSink 多路输出
type MultiSink struct { ... }
```

### Formatter 设计

```go
// interface/formatter.go
package iface

// Formatter 格式化接口，将事件转换为可展示的文本
type Formatter interface {
    FormatAgentOutput(content string) string
    FormatToolCall(name, args string) string
    FormatToolResult(name, result string, success bool) string
    FormatError(err error) string
    FormatApprovalRequest(toolName, args string) string
}

// CLFormatter CLI 格式化器
type CLIFormatter struct{}

func (f *CLIFormatter) FormatAgentOutput(content string) string {
    return content
}

func (f *CLIFormatter) FormatToolCall(name, args string) string {
    return fmt.Sprintf("🔧 [tool call] -> %s: %s\n", name, truncate(args, 200))
}

// ... 其他格式化方法

// WebFormatter Web 格式化器（可选）
type WebFormatter struct{}
```

### Approval 处理（移动到 interface 包）

```go
// interface/approval.go
package iface

import "aimc-go/assistant/core"

// ApprovalHandler 审批处理器
type ApprovalHandler interface {
    RequestApproval(event core.InterruptEvent) (bool, error)
}

// CLIApprovalHandler CLI 审批
type CLIApprovalHandler struct {
    scanner   *bufio.Scanner
    formatter Formatter
    sink      Sink
}

func (h *CLIApprovalHandler) RequestApproval(event core.InterruptEvent) (bool, error) {
    // 1. 格式化审批信息
    // 2. 输出到 sink
    // 3. 阻塞等待用户输入
    // 4. 返回审批结果
}

// SSEApprovalHandler SSE 审批
type SSEApprovalHandler struct {
    pending map[string]chan bool
    mu      sync.Mutex
}

func (h *SSEApprovalHandler) RequestApproval(event core.InterruptEvent) (bool, error) {
    // 1. 创建 channel
    // 2. 注册到 pending
    // 3. 阻塞等待 HTTP 回调
}

func (h *SSEApprovalHandler) Submit(interruptID string, approved bool) {
    // HTTP handler 调用，解除阻塞
}
```

### CLI 交互层

```go
// interface/cli/cli.go
package cli

import (
    "aimc-go/assistant/core"
    iface "aimc-go/assistant/interface"
)

func Cli() {
    ctx := context.Background()
    
    // 创建组件
    formatter := &iface.CLIFormatter{}
    sink := iface.NewStdoutSink()
    approvalHandler := &iface.CLIApprovalHandler{...}
    
    // 创建事件处理器
    eventHandler := func(event core.Event) {
        switch event.Type {
        case core.EventAgentOutput:
            data := event.Data.(core.AgentOutputEvent)
            sink.Emit(iface.Chunk{
                Kind:    iface.KindAssistant,
                Content: formatter.FormatAgentOutput(data.Content),
            })
        case core.EventToolCall:
            data := event.Data.(core.ToolCallEvent)
            sink.Emit(iface.Chunk{
                Kind:    iface.KindToolCall,
                Content: formatter.FormatToolCall(data.ToolName, data.Arguments),
            })
        case core.EventInterrupt:
            data := event.Data.(core.InterruptEvent)
            approved, _ := approvalHandler.RequestApproval(data)
            // 处理审批结果
        }
    }
    
    // 创建 Agent 和 Runner
    agent := core.NewAgent(ctx)
    runner := core.NewRunner(agent, eventHandler, core.WithStore(store))
    
    // 交互循环
    for {
        // 读取输入
        // 运行 runner.Run()
    }
}
```

### 入口文件

```go
// cli.go（保持不变，但只作为入口）
package assistant

import "aimc-go/assistant/interface/cli"

func Cli() {
    cli.Cli()
}
```

---

## 实现任务

### Task 1: 创建新目录结构和事件类型定义

**Files:**
- Create: `assistant/core/event.go`
- Create: `assistant/core/event_handler.go`
- Create: `assistant/interface/sink.go`
- Create: `assistant/interface/formatter.go`
- Create: `assistant/interface/approval.go`
- Create: `assistant/interface/cli/cli.go`

**Step 1: 创建目录结构**

```bash
mkdir -p assistant/core
mkdir -p assistant/interface/cli
mkdir -p assistant/interface/sse
```

**Step 2: 创建 core/event.go**

定义所有事件类型，确保是纯数据结构，不依赖任何外部包。

**Step 3: 创建 core/event_handler.go**

实现 Eino AgentEvent 到结构化 Event 的转换逻辑。

**Step 4: 创建 interface/sink.go**

移动 sink 包的内容到 interface 包，保持接口不变。

**Step 5: 创建 interface/formatter.go**

实现 CLI 格式化器，将 emoji 和格式化逻辑从核心层移出。

**Step 6: 创建 interface/approval.go**

移动 approval 包的内容到 interface 包，修改为使用 Formatter 接口。

**Step 7: 创建 interface/cli/cli.go**

实现 CLI 交互层，组装所有组件。

---

### Task 2: 重构 core/runner.go

**Files:**
- Create: `assistant/core/runner.go`
- Create: `assistant/core/agent.go`

**Step 1: 创建 core/agent.go**

从 agent/agent.go 迁移 Agent 构建逻辑，移除与展示相关的代码。

**Step 2: 创建 core/runner.go**

重构 Runner：
- 移除 Sink 依赖
- 添加 EventHandler 回调
- 实现事件转换逻辑

**Step 3: 验证编译**

```bash
cd /home/lsk/projects/aimc-pro/aimc-go
go build ./assistant/...
```

---

### Task 3: 更新 store 包

**Files:**
- Modify: `assistant/store/store.go`

**Step 1: 添加工厂函数**

将 `agent.JSONLStore()` 移动到 store 包：

```go
func NewJSONLStore(dir string) Store {
    return &JSONLStore{dir: dir, sessions: make(map[string]*Session)}
}
```

**Step 2: 验证编译**

```bash
go build ./assistant/store/...
```

---

### Task 4: 重构 command 包

**Files:**
- Modify: `assistant/command/command.go`

**Step 1: 解耦 Runner 依赖**

将 `*agent.Runner` 改为接口：

```go
// Runner 接口，解耦 command 包对 agent 包的依赖
type Runner interface {
    Run(ctx context.Context, sessionID, query string) error
    Resume(ctx context.Context, sessionID string, approved bool) error
}
```

**Step 2: 更新 Dependencies**

```go
type Dependencies struct {
    Store     store.Store
    Runner    Runner  // 接口而不是具体类型
    SessionID *string
    Scanner   *bufio.Scanner
}
```

---

### Task 5: 更新 cli.go 入口

**Files:**
- Modify: `assistant/cli.go`

**Step 1: 简化入口**

```go
package assistant

import "aimc-go/assistant/interface/cli"

func Cli() {
    cli.Cli()
}
```

**Step 2: 验证编译和运行**

```bash
go build ./assistant/...
go run . assistant
```

---

### Task 6: 清理旧代码

**Files:**
- Delete: `assistant/agent/sink.go` (如果存在)
- Delete: `assistant/agent/runner_event.go` (逻辑已迁移到 core/)
- Modify: `assistant/agent/runner.go` (移除 Sink 依赖)
- Delete: `assistant/sink/` 目录
- Delete: `assistant/approval/` 目录

**Step 1: 确认新代码工作正常**

```bash
go run . assistant
```

**Step 2: 删除旧文件**

```bash
rm -rf assistant/sink
rm -rf assistant/approval
rm assistant/agent/runner_event.go
```

**Step 3: 更新 agent/runner.go**

移除 Sink 相关代码，确保只依赖 core 包。

---

### Task 7: 实现 SSE Handler（可选）

**Files:**
- Create: `assistant/interface/sse/handler.go`
- Create: `assistant/interface/sse/sse_sink.go`

**Step 1: 实现 SSESink**

```go
package sse

import (
    "fmt"
    "net/http"
    iface "aimc-go/assistant/interface"
)

type SSESink struct {
    w       http.ResponseWriter
    flusher http.Flusher
}

func NewSSESink(w http.ResponseWriter) (iface.Sink, error) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        return nil, fmt.Errorf("streaming not supported")
    }
    
    return &SSESink{w: w, flusher: flusher}, nil
}

func (s *SSESink) Emit(c iface.Chunk) {
    fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", c.Kind, c.Content)
    s.flusher.Flush()
}
```

**Step 2: 实现 HTTP Handler**

```go
func ChatHandler(w http.ResponseWriter, r *http.Request) {
    // 1. 创建 SSE sink
    // 2. 创建 approval handler
    // 3. 创建 event handler
    // 4. 创建 runner
    // 5. 运行对话
}
```

---

## 依赖关系图

```
                    ┌─────────────────┐
                    │   cli.go        │  入口
                    └────────┬────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        interface/                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌───────────┐ │
│  │ cli/cli.go  │  │ sse/        │  │ formatter.go│  │ approval  │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬─────┘ │
│         │                │                │                │       │
│         │                │                │                │       │
│         └────────────────┴────────────────┴────────────────┘       │
│                          │                                          │
│                          │ Event                                    │
│                          │                                          │
│                   ┌──────┴──────┐                                   │
│                   │   sink.go   │                                   │
│                   └─────────────┘                                   │
└─────────────────────────────────────────────────────────────────────┘
                             │
                             │ 依赖
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                           core/                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌───────────┐ │
│  │  agent.go   │  │  runner.go  │  │  event.go   │  │ handler   │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬─────┘ │
│         │                │                │                │       │
│         └────────────────┴────────────────┴────────────────┘       │
│                          │                                          │
└──────────────────────────┼──────────────────────────────────────────┘
                           │
                           │ 依赖
                           ▼
                    ┌─────────────┐
                    │  Eino ADK   │
                    └─────────────┘
```

---

## 验证清单

- [ ] `go build ./assistant/...` 编译通过
- [ ] `go run . assistant` CLI 交互正常
- [ ] 事件正确传递到 formatter
- [ ] 审批流程正常工作
- [ ] 命令系统（/new, /resume, /quit）正常
- [ ] 会话存储正常
- [ ] SSE handler 可选实现（如果需要）

---

## 执行方式

**推荐：Subagent-Driven（当前会话）**

每个 Task 独立执行，完成后 review 再继续下一个。
