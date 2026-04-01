# 提取 Agent/Runner 默认配置为独立函数 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** agent.New / NewRunner 不再内置默认值，提供独立的默认值函数供调用方显式选择。

**Design:**
- `agent.New` 校验所有依赖非空，不做任何填充
- 提供 `agent.PresetTools(cm)`, `agent.PresetMiddlewares(ctx, cm)` 方便调用方
- `NewRunner` 校验 store/sink 非空，不做任何填充
- 提供 `agent.JSONLStore(dir)`, `agent.StdoutSink()` 方便调用方
- 保留：MaxIterations 默认 30、`llm.DefaultConfig()`

---

## Task 1: agent.go — 移除内部默认值，抽取独立默认函数

**Files:**
- Modify: `assistant/agent/agent.go`

**Step 1: 重写 agent.go**

```go
package agent

import (
	"aimc-go/assistant/agent/middleware"
	"aimc-go/assistant/agent/tools"
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

type AgentConfig struct {
	Name          string
	Description   string
	Instruction   string
	MaxIterations int // 0 defaults to 30

	Model       model.ToolCallingChatModel     // required
	Tools       []tool.BaseTool                // required
	Middlewares []adk.ChatModelAgentMiddleware // required
}

// PresetTools returns the built-in tools (RAG + search).
func PresetTools(cm model.BaseChatModel) ([]tool.BaseTool, error) {
	return tools.InitTools(cm)
}

// PresetMiddlewares returns the built-in infra middlewares (patch, summarization, reduction, filesystem, skill).
func PresetMiddlewares(ctx context.Context, cm model.BaseChatModel, cfg middleware.Config) ([]adk.ChatModelAgentMiddleware, error) {
	return middleware.SetupMiddlewares(ctx, cm, cfg)
}

func New(ctx context.Context, cfg AgentConfig) (adk.Agent, error) {
	if cfg.Model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if len(cfg.Tools) == 0 {
		return nil, fmt.Errorf("tools is required, use agent.PresetTools(cm) for built-in tools")
	}
	if len(cfg.Middlewares) == 0 {
		return nil, fmt.Errorf("middlewares is required, use agent.PresetMiddlewares(ctx, cm, middleware.Config{}) for built-in middlewares")
	}
	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 30
	}

	ag, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Instruction:   cfg.Instruction,
		MaxIterations: cfg.MaxIterations,
		Model:         cfg.Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.Tools,
			},
		},
		Handlers: cfg.Middlewares,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return ag, nil
}
```

**Step 2: 编译检查**

Run: `go build ./assistant/agent/...`

Expected: 编译失败（assistant.go 使用旧 API）

**Step 3: Commit**

```bash
git add assistant/agent/agent.go
git commit -m "refactor(agent): extract defaults to PresetTools/PresetMiddlewares, require explicit injection"
```

---

## Task 2: runner.go — 移除内部默认值，抽取独立默认函数

**Files:**
- Modify: `assistant/agent/runner.go`

**Step 1: 重写 runner.go**

```go
package agent

import (
	"aimc-go/assistant/sink"
	"aimc-go/assistant/store"
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

type Runner struct {
	inner   *adk.Runner
	handler *EventHandler
	store   store.Store
	sink    sink.Sink
}

type RunnerOption func(*Runner)

func WithStore(s store.Store) RunnerOption {
	return func(r *Runner) {
		r.store = s
	}
}

func WithSink(s sink.Sink) RunnerOption {
	return func(r *Runner) {
		r.sink = s
	}
}

// JSONLStore returns a JSONL store at the given directory.
func JSONLStore(dir string) store.Store {
	return &store.JSONLStore{
		Dir:   dir,
		Cache: make(map[string]*store.Session),
	}
}

// StdoutSink returns a stdout sink.
func StdoutSink() sink.Sink {
	return &sink.StdoutSink{}
}

func NewRunner(agent adk.Agent, opts ...RunnerOption) (*Runner, error) {
	r := &Runner{
		inner: adk.NewRunner(context.Background(), adk.RunnerConfig{
			Agent:           agent,
			EnableStreaming: true,
		}),
		handler: &EventHandler{},
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.store == nil {
		return nil, fmt.Errorf("store is required, use agent.JSONLStore(dir) for a JSONL store")
	}
	if r.sink == nil {
		return nil, fmt.Errorf("sink is required, use agent.StdoutSink() for stdout")
	}

	return r, nil
}

func (r *Runner) Run(ctx context.Context, sessionID, query string) (string, error) {
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	session, _ := r.store.GetOrCreate(ctx, sessionID)
	_ = r.store.Append(ctx, session.ID, schema.UserMessage(query))

	history := session.Messages
	iter := r.inner.Run(ctx, history)
	content, err := r.processEventStream(ctx, iter)
	if err != nil {
		return "", err
	}

	_ = r.store.Append(ctx, session.ID, schema.AssistantMessage(content, nil))
	return content, nil
}

func (r *Runner) processEventStream(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	ec := &EventContext{
		Ctx:       ctx,
		Collector: &strings.Builder{},
		Sink:      r.sink,
	}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		err := r.handler.HandleEvent(ec, event)
		if err != nil {
			return "", err
		}
	}

	return ec.Collector.String(), nil
}
```

**Step 2: 编译检查**

Run: `go build ./assistant/agent/...`

Expected: 编译失败（assistant.go 使用旧 API）

**Step 3: Commit**

```bash
git add assistant/agent/runner.go
git commit -m "refactor(runner): extract defaults to JSONLStore/StdoutSink, require explicit injection"
```

---

## Task 3: assistant.go — 显式调用默认函数

**Files:**
- Modify: `assistant/assistant.go`

**Step 1: 重写 assistant.go**

```go
package assistant

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/agent/llm"
	"aimc-go/assistant/agent/middleware"
	"aimc-go/assistant/agent/prompts"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

func Agent() {
	ctx := context.Background()

	// 1. model
	cm, err := llm.NewChatModel(ctx, llm.DefaultConfig())
	if err != nil {
		panic(err)
	}

	// 2. tools — 使用默认工具集
	agentTools, err := agent.PresetTools(cm)
	if err != nil {
		panic(err)
	}

	// 3. middlewares — 使用默认中间件
	middlewares, err := agent.PresetMiddlewares(ctx, cm, middleware.Config{})
	if err != nil {
		panic(err)
	}

	// 4. agent
	projectRoot := "/home/lsk/projects/eino-demo"
	ag, err := agent.New(ctx, agent.AgentConfig{
		Name:          "enio-assistant",
		Description:   "enio tutorial assistant",
		Instruction:   fmt.Sprintf(prompts.EinoTutorial, projectRoot, projectRoot, projectRoot, projectRoot),
		Model:         cm,
		Tools:         agentTools,
		Middlewares:   middlewares,
		MaxIterations: 30,
	})
	if err != nil {
		panic(err)
	}

	// 5. runner — 使用默认 store/sink
	runner, err := agent.NewRunner(ag,
		agent.WithStore(agent.JSONLStore("./data/sessions")),
		agent.WithSink(agent.StdoutSink()),
	)
	if err != nil {
		panic(err)
	}

	sessionID := uuid.New().String()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		_, _ = fmt.Fprint(os.Stdout, "you> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		_, err = runner.Run(ctx, sessionID, line)
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

**Step 2: 全量编译**

Run: `go build ./assistant/...`

Expected: PASS

**Step 3: Commit**

```bash
git add assistant/assistant.go
git commit -m "refactor(assistant): use explicit default functions, no hidden dependencies"
```

---

## 最终 API 用法

```go
// 快速启动（使用框架默认配置）
cm, _ := llm.NewChatModel(ctx, llm.DefaultConfig())
agentTools, _ := agent.PresetTools(cm)
middlewares, _ := agent.PresetMiddlewares(ctx, cm, middleware.Config{})
ag, _ := agent.New(ctx, agent.AgentConfig{
    Model: cm, Tools: agentTools, Middlewares: middlewares,
})
runner, _ := agent.NewRunner(ag,
    agent.WithStore(agent.JSONLStore("./data")),
    agent.WithSink(agent.StdoutSink()),
)

// 完全自定义（不用任何 Default 函数）
myTools := []tool.BaseTool{myCustomTool1, myCustomTool2}
myMWs := []adk.ChatModelAgentMiddleware{myMW1, myMW2}
ag, _ := agent.New(ctx, agent.AgentConfig{
    Model: cm, Tools: myTools, Middlewares: myMWs,
})
runner, _ := agent.NewRunner(ag,
    agent.WithStore(myRedisStore),
    agent.WithSink(mySSESink),
)
```
