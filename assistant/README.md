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
│  │ CLITransport    │              │    SSEHub           │    │
│  │ - stdin scanner │              │  - Acquire/Release  │    │
│  │ - stdout 输出   │              │  - SubmitApproval() │    │
│  └─────────────────┘              └─────────────────────┘    │
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
│  │  - ID, Transport                                       │  │
│  │  - Emit() - 向客户端推送事件                           │  │
│  │  - WaitInput() - 阻塞等待输入（审批/用户干预）         │  │
│  │  - Close() - 关闭传输层                                │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│                       Infrastructure Layer                   │
│                                                              │
│  Transport: CLITransport, SSETransport, MultiTransport       │
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
│   ├── runtime.go              # Runtime 结构 + New/Run/Resume/Generate/Events + 工具函数
│   └── events.go               # 事件消费管道 + 消息处理 + 审批处理
│
├── server/                    # 入口层
│   ├── bootstrap.go            # NewRuntime 组装
│   ├── cli.go                  # RunCLI
│   ├── sse.go                  # SSEModule + SSEHub
│   └── sse.html                # SSE 测试页面
│
├── session/                   # 会话数据结构 & 传输层
│   ├── session.go              # Session + New
│   ├── event.go                # Event + EventType
│   └── transport.go            # Transport 接口 + CLITransport/SSETransport/MultiTransport
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

### Transport（传输层抽象）

Transport 统一了 session 的输入输出，每种实现对应一种传输方式，内部持有该方式所需的全部资源。

```go
type Transport interface {
    Emit(Event) error                                // 向客户端推送事件
    WaitInput(ctx context.Context) (InputEvent, error) // 阻塞等待客户端输入
    Close()                                          // 关闭传输层，释放资源
}
```

**内置实现：**

| 实现 | 输出 | 输入 | 场景 |
|------|------|------|------|
| `CLITransport` | stdout | stdin scanner | 终端交互 |
| `SSETransport` | HTTP SSE 帧 | channel（Submit 注入） | Web 推送 |
| `MultiTransport` | 广播到所有子 Transport | 委托给第一个 | 多路输出 |

```go
// CLI
transport := session.NewCLITransport(scanner)
sess := session.New(sessionID, transport)

// SSE
transport := session.NewSSETransport(ctx, w, flusher)
sess := session.New(sessionID, transport)
```

扩展新传输方式只需实现 `Transport` 接口。

### Session（会话容器）

Session 是一轮会话的交互容器，持有 ID 和 Transport。

```go
type Session struct {
    ID        string
    Transport Transport
}
```

**关键方法：**
- `Emit(event)` - 向客户端推送事件（委托 Transport）
- `WaitInput(ctx)` - 阻塞等待输入（委托 Transport）
- `Close()` - 关闭传输层（委托 Transport）

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

### Event（事件模型）

```go
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

## 数据流

### CLI 流程

```
用户输入 (stdin)
       │
       ▼
  CLITransport(scanner)
       │
       ▼
     Session
       │
       ▼
     Runtime.Run(ctx, session, query)
       │
       ├──→ drain() → session.Emit() → CLITransport.Emit() → stdout
       │
       └→ handleInterrupt() → session.WaitInput() → CLITransport.WaitInput()
       
     返回循环等待下一轮输入
```

### SSE 流程

```
HTTP POST /chat
       │
       ▼
  SSEHub.Acquire(sessionID, transport)
       │
       ▼
     Session (Transport: SSETransport)
       │
       ▼
     Runtime.Run() (goroutine)
       │
       └──→ session.Emit() → SSETransport.Emit() → SSE 推送

HTTP POST /approval
       │
       ▼
  SSEHub.SubmitApproval()
       │
       ▼
     SSETransport.Submit() ─── 解除 WaitInput 阻塞
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
| 新交互模式（如 WebSocket） | 实现 Transport 接口，在 server 包添加对应的 Hub |
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

1. **Transport 统一输入输出**，每种传输方式实现一个 Transport，内聚该方式的全部 I/O 逻辑。
2. **Session = 会话**，不是单轮对话。Session 在多轮对话期间保持。
3. **Runtime 统一生命周期**，CLI 和 SSE 差异封装在 server 包。
4. **包职责清晰**，session 包只定义传输层抽象和会话结构，types 包放跨层领域类型，server 包管理入口和连接。
5. **AgentRunOption 透传**，Runtime 的 Run/Generate/Events 接受 `...adk.AgentRunOption`，允许调用方注入回调、修改模型参数等。
