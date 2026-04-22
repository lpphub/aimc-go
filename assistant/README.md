# Assistant - AI Agent Service Layer

基于 [eino](https://github.com/cloudwego/eino) 的 AI Agent 服务层，支持 CLI 和 SSE 双交互模式。

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                     Application Layer                        │
│                                                              │
│  ┌─────────────────┐              ┌─────────────────────┐    │
│  │    CLI Entry    │              │    SSE Module       │    │
│  │ (server/cli.go) │              │  (server/sse.go)    │    │
│  └─────────────────┘              └─────────────────────┘    │
│           │                                 │                │
│           ▼                                 ▼                │
│  ┌─────────────────┐              ┌─────────────────────┐    │
│  │   NewCLI()      │              │    SSEHub           │    │
│  │  - OnInput 回调 │              │  - Acquire/Release  │    │
│  │  - StdoutWriter │              │  - SubmitApproval() │    │
│  └─────────────────┘              │  - SSEWriter        │    │
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
│  │  - ID, Writer, Input, OnInput                         │  │
│  │  - WaitInput() - 阻塞等待输入（审批/用户干预）         │  │
│  │  - Write(), Close()                                   │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│                       Infrastructure Layer                   │
│                                                              │
│  Writer: StdoutWriter, SSEWriter                             │
│  Store: JSONLStore                                           │
│  Approval: Info, Result                                      │
│  Agent: agent.New()                                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 文件结构

```
assistant/
├── cmd/
│   └── agent/main.go          # CLI 入口（调用 server.RunCLI）
│
├── server/                    # 入口层
│   ├── cli.go                 # RunCLI + NewCLI builder
│   ├── sse.go                 # SSEModule + SSEHub
│   └── runtime.go             # NewRuntime 创建
│
├── session/                   # 数据结构层
│   ├── session.go             # Session + New(withChan) + 方法
│   ├── input.go               # InputEvent + InputType
│   ├── chunk.go               # Chunk + ChunkType
│   └── writer.go              # Writer 接口 + 实现
│
├── runtime/                   # 运行时核心
│   ├── runtime.go             # Runtime 结构 + Run/Resume
│   └── event.go               # EventHandler
│
├── store/                     # 存储层
│   ├── store.go               # Store 接口
│   └── jsonl.go               # JSONLStore 实现
│
├── approval/                  # 审批模块
│   └── approval.go            # Info + Result
│
├── agent/                     # Agent 配置
│   ├── agent.go               # Agent 创建
│   ├── config.go              # Config + 函数式选项
│   ├── llm/                   # LLM 提供者
│   ├── middleware/            # 中间件
│   ├── tools/                 # 工具
│   └── prompts/               # 提示词
│
└── README.md
```

## 核心组件

### Session（双向交互容器）

Session 是一轮会话的交互容器，封装输出通道和输入通道。

```go
type Session struct {
    ID        string
    Writer    Writer              // 输出通道
    Input     chan InputEvent     // withChan=true: channel 输入
    OnInput   func(...)           // withChan=false: 阻塞回调
}

// withChan=true: channel 输入（SSE/WebSocket）
session.New(sessionID, writer, true)

// withChan=false: 阻塞回调 OnInput（CLI）
session.New(sessionID, writer, false)
```

**关键方法：**
- `WaitInput(ctx)` - 阻塞等待输入（审批结果或用户干预）
- `Write(chunk)` - 输出消息片段
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

### Writer（单向输出接口）

Writer 是单向输出接口，实现必须是并发安全的。

```go
type Writer interface {
    Write(Chunk) error
}

type Chunk struct {
    Type    ChunkType      `json:"type"`
    Content string         `json:"content"`
    Meta    map[string]any `json:"meta,omitempty"`
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

## 数据流

### CLI 流程

```
用户输入 (stdin)
       │
       ▼
  server.NewCLI(sessionID, StdoutWriter, scanner)
       │
       ▼
     Session (OnInput: 读 stdin)
       │
       ▼
     Runtime.Run(ctx, session, query)
       │
       ├──→ EventHandler.Drain() → session.Write() → StdoutWriter
       │
       └→ handleInterrupt() → session.WaitInput() → OnInput()
       
     返回循环等待下一轮输入
```

### SSE 流程

```
HTTP POST /chat
       │
       ▼
  SSEHub.Acquire(sessionID, writer, flusher)
       │
       ▼
     Session (Input: channel)
       │
       ▼
     Runtime.Run() (goroutine)
       │
       └──→ session.Write() → SSEWriter → SSE 推送

HTTP POST /approval
       │
       ▼
  SSEHub.SubmitApproval()
       │
       ▼
     Input ←─── 解除 WaitInput 阻塞
       │
       ▼
  Runtime.Resume() → 继续执行
```

## 使用示例

### CLI 模式

```go
func main() {
    server.RunCLI()
}

// 内部实现（server/cli.go）
func RunCLI() {
    rt, _ := NewRuntime()
    ctx := context.Background()
    scanner := bufio.NewScanner(os.Stdin)
    sessionID := uuid.New().String()
    
    sess := NewCLI(sessionID, session.NewStdoutWriter(), scanner)
    
    for {
        fmt.Print("👤: ")
        if !scanner.Scan() {
            break
        }
        query := strings.TrimSpace(scanner.Text())
        
        rt.Run(ctx, sess, query)
    }
}
```

### SSE 模式

```go
// 初始化模块
module, _ := server.NewSSE()

// HTTP Handler（server/sse.go 已实现）
func (m *SSEModule) Routes(r *gin.RouterGroup) {
    assistant := r.Group("/assistant")
    assistant.GET("", m.ssePage)       // SSE 页面
    assistant.POST("/chat", m.chat)    // 聊天接口
    assistant.POST("/approval", m.approval)  // 审批接口
}

// 内部实现
func (m *SSEModule) chat(c *gin.Context) {
    // 设置 SSE headers
    c.Header("Content-Type", "text/event-stream")
    
    sess, _ := m.hub.Acquire(ctx, sessionID, c.Writer, flusher)
    
    go func() {
        defer sess.Close()
        defer m.hub.Release(sessionID)
        m.rt.Run(ctx, sess, query)
    }()
    
    <-ctx.Done()
}

func (m *SSEModule) approval(c *gin.Context) {
    var req ApprovalRequest
    json.NewDecoder(c.Request.Body).Decode(&req)
    
    m.hub.SubmitApproval(req.SessionID, &approval.Result{
        ApprovalID: req.ApprovalID,
        Approved:   req.Approved,
    })
}
```

## 扩展点

| 扩展需求 | 实现方式 |
|---------|---------|
| 新交互模式（如 WebSocket） | 实现 Writer 接口，在 server 包添加对应的 Hub 和 builder |
| 新输出格式（如 WebSocket） | 实现 Writer 接口 |
| 新存储后端（如 Redis） | 实现 Store 接口 |
| 新消息类型 | 添加 ChunkType 常量，在 EventHandler 中处理 |
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
2. **Writer 单向简单**，不承担审批逻辑，只做输出。
3. **Runtime 统一生命周期**，CLI 和 SSE 差异封装在 server 包。
4. **并发安全**，Writer 实现必须线程安全，Session.Close 使用 sync.Once。
5. **包职责清晰**，session 包只定义数据结构，server 包管理入口和连接。