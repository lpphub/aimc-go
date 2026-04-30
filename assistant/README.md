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
│  │  - Run(ctx, session, query, ...AgentRunOption)         │  │
│  │  - Resume(ctx, session, checkpointID, resumeData)      │  │
│  │  - Generate(ctx, messages, ...AgentRunOption)         │  │
│  │  - Events(ctx, messages, ...AgentRunOption)           │  │
│  │                                                        │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌───────────────┐  │  │
│  │  │   Store     │  │    Agent    │  │ CheckPointStore│  │  │
│  │  │   (注入)    │  │   (注入)    │  │    (内置)      │  │  │
│  │  └─────────────┘  └─────────────┘  └───────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                       Session                          │  │
│  │  - ID, Sink, InputChan, OnInput                        │  │
│  │  - WaitInput() - 阻塞等待输入（审批/用户干预）         │  │
│  │  - Emit(), Close()                                    │  │
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
│  Types: ApprovalInfo, ApprovalResult                         │
│  Agent: agent.New()                                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 文件结构

```
assistant/
├── agent/                     # Agent 配置与组装
│   ├── agent.go               # Agent 创建入口
│   ├── config.go              # Config + 函数式选项
│   ├── llm/                   # LLM 提供者
│   │   └── provider.go
│   ├── middleware/             # 中间件
│   │   ├── init.go             # 中间件组装
│   │   ├── builtin.go          # 内置中间件（summarization, filesystem 等）
│   │   ├── approval.go         # 审批中间件
│   │   └── toolrecovery.go     # 工具错误恢复中间件
│   ├── tools/                  # 工具
│   │   ├── init.go             # 工具注册
│   │   ├── rag.go              # RAG 文档问答工具
│   │   └── time.go             # 当前时间工具
│   ├── prompts/                # 提示词模板
│   │   └── template.go
│   └── callback/               # 用量统计
│       └── usage.go
│
├── runtime/                   # 运行时核心
│   ├── runtime.go              # Runtime 结构 + New/Run/Resume/Generate/Events
│   ├── events.go               # 事件消费管道（drain, handleEvent, handleAction）
│   ├── handler.go              # 消息处理 + 审批处理
│   └── utils.go                # trimRounds, truncate
│
├── server/                    # 入口层
│   ├── bootstrap.go            # NewRuntime 组装
│   ├── cli.go                  # RunCLI + NewCLI
│   ├── sse.go                  # SSEModule + SSEHub
│   └── sse.html                # SSE 测试页面
│
├── session/                   # 会话数据结构 & I/O 通道
│   ├── session.go              # Session + New(withChan)
│   ├── input.go                # InputEvent + InputType
│   ├── event.go                # Event + EventType
│   └── sink.go                 # Sink 接口 + StdoutSink/SSESink/MultiSink
│
├── store/                     # 消息持久化
│   ├── store.go                # Store 接口
│   └── jsonl.go                # JSONLStore 实现
│
├── types/                     # 跨层共享领域类型
│   └── approval.go             # ApprovalInfo + ApprovalResult
│
└── README.md
```

## 核心组件

### Session（双向交互容器）

Session 是一轮会话的交互容器，封装输出通道和输入通道。

```go
type Session struct {
    ID        string
    Sink      Sink               // 输出通道
    InputChan chan InputEvent     // withChan=true: channel 输入
    OnInput   func(...)           // withChan=false: 阻塞回调
}

// withChan=true: channel 输入（SSE/WebSocket）
session.New(sessionID, sink, true)

// withChan=false: 阻塞回调 OnInput（CLI）
session.New(sessionID, sink, false)
```

**关键方法：**
- `WaitInput(ctx)` - 阻塞等待输入（审批结果或用户干预）
- `Emit(event)` - 发送输出事件
- `Close()` - 关闭会话（线程安全）

### Runtime（服务层核心）

Runtime 是服务层核心，管理完整会话生命周期。

```go
type Runtime struct {
    runner          *adk.Runner
    store           store.Store
    checkpointStore adk.CheckPointStore
    maxRounds       int
}
```

**关键方法：**
- `Run(ctx, session, query, ...AgentRunOption)` - 执行一轮对话
- `Resume(ctx, session, checkpointID, resumeData)` - 恢复中断的对话
- `Generate(ctx, messages, ...AgentRunOption)` - 原始 API，返回事件迭代器
- `Events(ctx, messages, ...AgentRunOption)` - 便利 API，返回事件 channel

`AgentRunOption` 透传至 `adk.Runner`，支持 `WithCallbacks`、`WithChatModelOptions`、`WithHistoryModifier` 等。

### Sink（事件输出接口）

Sink 是事件输出接口，遵循 Go Handler 惯例，实现必须是并发安全的。

```go
type Sink interface {
    Handle(Event) error
}

type Event struct {
    Type    EventType      `json:"type"`
    Content string         `json:"content"`
    Meta    map[string]any `json:"meta,omitempty"`
}
```

**EventType 类型：**
| 类型 | 说明 |
|------|------|
| `TypeAssistant` | AI 助手回复内容 |
| `TypeReasoning` | 思考过程 |
| `TypeToolCall` | 工具调用请求 |
| `TypeToolResult` | 工具执行结果 |
| `TypeMessage` | 系统消息 |
| `TypeApproval` | 审批请求 |
| `TypeApprovalRes` | 审批结果通知 |
| `TypeError` | 错误信息 |

## 数据流

### CLI 流程

```
用户输入 (stdin)
       │
       ▼
  server.NewCLI(sessionID, StdoutSink, scanner)
       │
       ▼
     Session (OnInput: 读 stdin)
       │
       ▼
     Runtime.Run(ctx, session, query)
       │
       ├──→ drain() → session.Emit() → StdoutSink
       │
       └→ handleInterrupt() → session.WaitInput() → OnInput()
       
     返回循环等待下一轮输入
```

### SSE 流程

```
HTTP POST /chat
       │
       ▼
  SSEHub.Acquire(sessionID, sink, flusher)
       │
       ▼
     Session (InputChan: channel)
       │
       ▼
     Runtime.Run() (goroutine)
       │
       └──→ session.Emit() → SSESink → SSE 推送

HTTP POST /approval
       │
       ▼
  SSEHub.SubmitApproval()
       │
       ▼
     InputChan ←─── 解除 WaitInput 阻塞
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
```

### SSE 模式

```go
// 初始化模块
module, _ := server.NewSSE()

// HTTP Handler（server/sse.go 已实现）
func (m *SSEModule) Routes(r *gin.RouterGroup) {
    assistant := r.Group("/assistant")
    assistant.GET("", m.ssePage)
    assistant.POST("/chat", m.chat)
    assistant.POST("/approval", m.approval)
}
```

### 直接使用 Runtime

```go
rt, _ := runtime.New(agent, runtime.WithStore(store), runtime.WithMaxRounds(30))

// 基本对话
rt.Run(ctx, sess, "hello")

// 带运行选项
rt.Run(ctx, sess, "hello", adk.WithCallbacks(handler))
```

## 扩展点

| 扩展需求 | 实现方式 |
|---------|---------|
| 新交互模式（如 WebSocket） | 实现 Sink 接口，在 server 包添加对应的 Hub 和 builder |
| 新输出格式 | 实现 Sink 接口 |
| 新存储后端（如 Redis） | 实现 Store 接口 |
| 新消息类型 | 添加 EventType 常量，在 events.go handleEvent() 中处理 |
| 自定义 Agent 配置 | 使用 `agent.WithProjectRoot()`, `agent.WithSkillDir()` 等选项 |
| 运行时回调/参数 | 透传 `adk.AgentRunOption` 到 `Run`/`Generate`/`Events` |

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
| `WithPlanTaskDir(path)` | 设置 Plan Task 目录 |
| `WithModel(m)` | 自定义 LLM 模型 |
| `WithTools(tools)` | 自定义工具集 |
| `WithMiddlewares(mw)` | 自定义中间件 |

## 设计原则

1. **Session = 会话**，不是单轮对话。Session 在多轮对话期间保持。
2. **Sink 单向简单**，不承担审批逻辑，只做输出。
3. **Runtime 统一生命周期**，CLI 和 SSE 差异封装在 server 包。
4. **并发安全**，Sink 实现必须线程安全，Session.Close 使用 sync.Once。
5. **包职责清晰**，session 包只定义 I/O 结构，types 包放跨层领域类型，server 包管理入口和连接。
6. **AgentRunOption 透传**，Runtime 的 Run/Generate/Events 接受 `...adk.AgentRunOption`，允许调用方注入回调、修改模型参数等。