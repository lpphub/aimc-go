# Assistant - AI Agent Service Layer

基于 [eino](https://github.com/cloudwego/eino) 的 AI Agent 服务层，支持 CLI 和 SSE 双交互模式。

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                     Application Layer                        │
│                                                              │
│  ┌─────────────────┐              ┌─────────────────────┐    │
│  │    CLI Entry    │              │    HTTP Handler     │    │
│  │   (main.go)     │              │   (api/handler.go)  │    │
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
│  │  - WaitInput() - 阻塞等待输入（审批/用户干预）         │  │
│  │  - Emit(), Collect(), Messages()                      │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│                       Infrastructure Layer                   │
│                                                              │
│  Sink: StdoutSink, SSESink                                   │
│  Store: JSONLStore                                           │
│  Approval: ApprovalInfo, ApprovalResult                      │
│  Agent: agent.New()                                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 文件结构

```
assistant/
├── main.go                    # CLI 入口

├── runtime/                   # 运行时核心
│   ├── runtime.go             # Runtime 结构 + Run/Resume 主逻辑
│   ├── event.go               # 事件处理方法（handleAgentEvent 等）
│   ├── session.go             # Session 结构定义 + WaitInput
│   ├── session_cli.go         # CLISessionBuilder 实现
│   └── session_sse.go         # SSESessionBuilder 实现

├── sink/                      # 输出层
│   ├── sink.go                # Sink 接口 + Chunk/ChunkType 定义
│   ├── stdout.go              # StdoutSink 实现（带并发安全）
│   └── sse.go                 # SSESink 实现（SSE JSON 推送）

├── store/                     # 存储层
│   ├── store.go               # Store 接口定义
│   └── jsonl.go               # JSONLStore 实现

├── approval/                  # 审批模块
│   └── approval.go            # ApprovalInfo + ApprovalResult 定义

├── agent/                     # Agent 配置
│   ├── agent.go               # Agent 创建入口（New + buildAgent）
│   ├── config.go              # Config 结构体 + 函数式选项
│   ├── llm/                   # LLM 提供者配置
│   ├── middleware/            # 中间件（审批、工具安全、内置中间件）
│   ├── tools/                 # 工具定义
│   └── prompts/               # 提示词模板

└── api/                       # HTTP 层
    └── handler.go             # SSE + Approval HTTP handlers
```

## 核心组件

### Session（双向交互容器）

Session 是一轮会话的交互容器，封装输出通道和输入通道。

```go
type Session struct {
    ID        string
    Sink      sink.Sink           // 输出通道
    InputChan chan InputEvent     // SSE: channel 输入
    OnInput   func(...)           // CLI: 阻塞回调
}
```

**关键方法：**
- `WaitInput(ctx)` - 阻塞等待输入（审批结果或用户干预）
- `Emit(chunk)` - 输出消息片段
- `Collect(msg)` - 收集消息用于持久化
- `Close()` - 关闭会话（线程安全）

### Runtime（服务层核心）

Runtime 是服务层唯一入口，管理完整会话生命周期。

```go
type Runtime struct {
    agent           adk.Agent
    store           store.Store
    checkpointStore adk.CheckPointStore
}
```

**关键方法：**
- `Run(ctx, session, query)` - 执行一轮对话
- `Resume(ctx, session, checkpointID, resumeData)` - 恢复中断的对话

### Sink（单向输出接口）

Sink 是单向输出接口，实现必须是并发安全的。

```go
type Sink interface {
    Emit(Chunk)
}

type Chunk struct {
    Type    ChunkType              // 消息类型
    Content string                 // 内容
    Meta    map[string]any         // 扩展元数据
}
```

**ChunkType 类型：**
| 类型 | 说明 |
|------|------|
| `TypeAssistant` | AI 助手回复内容 |
| `TypeToolCall` | 工具调用请求 |
| `TypeToolResult` | 工具执行结果 |
| `TypeMessage` | 系统消息 |
| `TypeApproval` | 审批请求 |
| `TypeApprovalRes` | 审批结果通知 |
| `TypeError` | 错误信息 |
| `TypeDone` | 对话结束信号 |

### SessionBuilder（交互层抽象）

SessionBuilder 让交互层决定如何创建 Session。

**CLI 场景：**
```go
builder := runtime.NewCLISessionBuilder(scanner)
session, _ := builder.Build(ctx, sessionID)
// OnInput 注册为阻塞读 stdin
```

**SSE 场景：**
```go
builder := runtime.NewSSESessionBuilder()
session, _ := builder.Build(ctx, sessionID, w, flusher)
// 不注册 OnInput，使用 InputChan
builder.SubmitApproval(sessionID, result)  // HTTP 回调写入
```

## 数据流

### CLI 流程

```
用户输入 (stdin)
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
       └──→ store.Append()

     返回循环等待下一轮输入
```

### SSE 流程

```
HTTP POST /chat
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
       │         └──→ session.Emit() → SSESink → SSE 推送
       │
       └──→ handleInterrupt()
                 │
                 ├──→ session.Emit(TypeApproval) → SSE 推送审批请求
                 └──→ session.WaitInput() → 阻塞等待 InputChan

HTTP POST /approval
       │
       ▼
  SubmitApproval()
       │
       ├──→ InputChan ←─── 解除 WaitInput 阻塞
       │
       ▼
  Runtime.Resume() → 继续执行
```

## 使用示例

### CLI 模式

```go
func main() {
    ctx := context.Background()
    agent, _ := agent.New(ctx)
    scanner := bufio.NewScanner(os.Stdin)
    
    builder := runtime.NewCLISessionBuilder(scanner)
    store := store.NewJSONLStore("./data/sessions")
    rt, _ := runtime.NewRuntime(agent, runtime.WithStore(store))
    
    session, _ := builder.Build(ctx, "session-123")
    defer session.Close()
    
    for {
        fmt.Print("👤: ")
        if !scanner.Scan() {
            break
        }
        query := strings.TrimSpace(scanner.Text())
        
        rt.Run(ctx, session, query)
    }
}
```

### SSE 模式

```go
// HTTP Handler
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    flusher, _ := w.(http.Flusher)
    
    session, _ := h.sessionBuilder.Build(r.Context(), sessionID, w, flusher)
    
    go func() {
        h.rt.Run(r.Context(), session, query)
        h.sessionBuilder.RemoveSession(sessionID)
        session.Close()
    }()
    
    <-r.Context().Done()
}

func (h *Handler) Approval(w http.ResponseWriter, r *http.Request) {
    var req ApprovalRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    h.sessionBuilder.SubmitApproval(req.SessionID, &approval.ApprovalResult{
        Approved: req.Approved,
    })
}
```

## 扩展点

| 扩展需求 | 实现方式 |
|---------|---------|
| 新交互模式（如 WebSocket） | 实现 SessionBuilder，决定使用 OnInput 或 InputChan |
| 新输出格式（如 WebSocket） | 实现 Sink 接口 |
| 新存储后端（如 Redis） | 实现 Store 接口 |
| 新消息类型 | 添加 ChunkType 常量，在 handleAgentEvent 中处理 |
| 自定义 Agent 配置 | 使用 `agent.WithProjectRoot()`, `agent.WithSkillDir()` 等选项 |

## Agent 配置

Agent 支持通过函数式选项配置：

```go
// 使用默认配置
ag, _ := agent.New(ctx)

// 自定义配置
ag, _ := agent.New(ctx,
    agent.WithProjectRoot("/path/to/project"),
    agent.WithSkillDir("/path/to/skills"),
    agent.WithModel(customModel),
    agent.WithTools(customTools),
    agent.WithMiddlewares(customMiddlewares),
)
```

**可用选项：**
| 选项 | 说明 |
|------|------|
| `WithProjectRoot(path)` | 设置项目根目录（用于 Prompt 模板中的绝对路径） |
| `WithSkillDir(path)` | 设置 Skill 目录（为空则不启用 Skill 中间件） |
| `WithModel(m)` | 自定义 LLM 模型 |
| `WithTools(tools)` | 自定义工具集 |
| `WithMiddlewares(mw)` | 自定义中间件 |

## 设计原则

1. **Session = 会话**，不是单轮对话。Session 在多轮对话期间保持。
2. **Sink 单向简单**，不承担审批逻辑，只做输出。
3. **Runtime 统一生命周期**，CLI 和 SSE 差异封装在 SessionBuilder。
4. **并发安全**，Sink 实现必须线程安全，Session.Close 使用 sync.Once。