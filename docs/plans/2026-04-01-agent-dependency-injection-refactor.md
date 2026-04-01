# Agent 依赖注入重构 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 agent/runner 的初始化从硬编码改为依赖注入，使用方决定 model/tools/middleware/store/sink 的创建方式。

**Architecture:** 将 `agent.New()` 从 god function 改为接收 `AgentConfig`（含依赖字段），提供 `defaults.go` 给开箱即用体验。Runner 用 Functional Options 模式注入 store/sink。

**Tech Stack:** Go, eino adk, functional options pattern

---

## Task 1: 重构 `llm/provider.go` — 移除硬编码 API key

**Files:**
- Modify: `assistant/agent/llm/provider.go`

**Step 1: 将常量改为 Config struct，保留默认值但允许覆盖**

```go
package llm

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

type Config struct {
	APIKey  string
	Model   string
	BaseURL string
}

func DefaultConfig() Config {
	return Config{
		APIKey:  "sk-sp-ac76140d6ae04e939ad6b82d71c2ea31",
		Model:   "glm-5",
		BaseURL: "https://coding.dashscope.aliyuncs.com/v1",
	}
}

func NewChatModel(ctx context.Context, cfg Config) (model.ToolCallingChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
		ByAzure: false,
	})
}
```

**Step 2: 验证编译通过**

Run: `cd /home/lsk/projects/aimc-pro/aimc-go && go build ./assistant/...`

Expected: 编译失败（因为调用方 `tools/tools.go` 和 `agent.go` 的 `llm.NewChatModel(ctx)` 签名变了）— 这是预期的，后续 task 修复。

**Step 3: Commit**

```bash
git add assistant/agent/llm/provider.go
git commit -m "refactor(llm): replace hardcoded constants with Config struct for dependency injection"
```

---

## Task 2: 重构 `tools/tools.go` — 接受外部 model，移除内部创建

**Files:**
- Modify: `assistant/agent/tools/tools.go`

**Step 1: 修改 `InitTools` 签名，接受 model 参数**

```go
package tools

import (
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

type Registry struct {
	tools []tool.BaseTool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make([]tool.BaseTool, 0),
	}
}

func (r *Registry) Register(t tool.BaseTool) {
	r.tools = append(r.tools, t)
}

func (r *Registry) GetAll() []tool.BaseTool {
	return r.tools
}

// InitTools 初始化所有内置工具，cm 用于需要模型的工具（如 RAG）
func InitTools(cm model.BaseChatModel) ([]tool.BaseTool, error) {
	registry := NewRegistry()

	ragTool, err := BuildRAGTool(nil, cm)
	if err != nil {
		return nil, fmt.Errorf("build rag tool: %w", err)
	}
	registry.Register(ragTool)

	searchTool, err := NewSearchTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create search tool: %w", err)
	}
	registry.Register(searchTool)

	return registry.GetAll(), nil
}
```

**Step 2: 验证编译**

Run: `cd /home/lsk/projects/aimc-pro/aimc-go && go build ./assistant/...`

Expected: 编译失败（`agent.go` 中调用 `InitTools()` 无参数）

**Step 3: Commit**

```bash
git add assistant/agent/tools/tools.go
git commit -m "refactor(tools): accept external model in InitTools, remove redundant model creation"
```

---

## Task 3: 重构 `middleware/infra.go` — 移除硬编码 skill 路径

**Files:**
- Modify: `assistant/agent/middleware/infra.go`

**Step 1: 将 hardcoded path 改为配置参数**

```go
package middleware

import (
	"context"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/patchtoolcalls"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/components/model"
)

// Config 中间件配置
type Config struct {
	SkillDir string // 技能文件目录，为空则不加载 skill middleware
}

func SetupMiddlewares(ctx context.Context, chatModel model.BaseChatModel, cfg Config) ([]adk.ChatModelAgentMiddleware, error) {
	middlewares, err := setupInfraMiddleware(ctx, chatModel, cfg)
	if err != nil {
		return nil, err
	}

	middlewares = append(middlewares, &safeToolMiddleware{})

	return middlewares, nil
}

func setupInfraMiddleware(ctx context.Context, chatModel model.BaseChatModel, cfg Config) ([]adk.ChatModelAgentMiddleware, error) {
	var middlewares []adk.ChatModelAgentMiddleware

	backend, err := local.NewBackend(ctx, &local.Config{})
	if err != nil {
		return nil, err
	}

	if patchMW, err := patch(ctx); err == nil {
		middlewares = append(middlewares, patchMW)
	}

	if sumMW, err := sum(ctx, chatModel); err == nil {
		middlewares = append(middlewares, sumMW)
	}

	if reductionMW, err := reduce(ctx, backend); err == nil {
		middlewares = append(middlewares, reductionMW)
	}

	if fsMW, err := fs(ctx, backend); err == nil {
		middlewares = append(middlewares, fsMW)
	}

	if cfg.SkillDir != "" {
		if skillMW, err := skills(ctx, backend, cfg.SkillDir); err == nil {
			middlewares = append(middlewares, skillMW)
		}
	}

	return middlewares, nil
}
```

**Step 2: 修改 `skills` 函数接受路径参数**

```go
func skills(ctx context.Context, backend *local.Local, skillDir string) (adk.ChatModelAgentMiddleware, error) {
	skillBackend, _ := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
		Backend: backend,
		BaseDir: skillDir,
	})
	skillMW, err := skill.NewMiddleware(ctx, &skill.Config{
		Backend: skillBackend,
	})
	if err != nil {
		return nil, err
	}
	return skillMW, nil
}
```

（`fs`, `patch`, `reduce`, `sum` 函数保持不变）

**Step 3: 验证编译**

Run: `cd /home/lsk/projects/aimc-pro/aimc-go && go build ./assistant/...`

Expected: 编译失败（`agent.go` 调用 `SetupMiddlewares` 签名变了）

**Step 4: Commit**

```bash
git add assistant/agent/middleware/infra.go
git commit -m "refactor(middleware): externalize skill dir config, remove hardcoded paths"
```

---

## Task 4: 重构 `agent.go` — AgentConfig 注入依赖

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
	"github.com/cloudwego/eino/compose"
)

type AgentConfig struct {
	// 元数据
	Name        string
	Description string
	Instruction string

	// 依赖（由调用方注入）
	Model       model.ToolCallingChatModel // 必须
	Tools       []adk.BaseTool             // 可选，传 nil 则不注册业务工具
	Middlewares []adk.ChatModelAgentMiddleware // 可选，传 nil 则使用默认中间件

	// 运行配置
	MaxIterations int
}

func New(ctx context.Context, cfg AgentConfig) (adk.Agent, error) {
	if cfg.Model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 30
	}

	// 如果未指定 tools，尝试用默认方式初始化（需要 model）
	if cfg.Tools == nil {
		defaultTools, err := tools.InitTools(cfg.Model)
		if err != nil {
			return nil, fmt.Errorf("init default tools: %w", err)
		}
		cfg.Tools = defaultTools
	}

	// 如果未指定 middlewares，使用默认基础设施中间件
	if cfg.Middlewares == nil {
		defaultMW, err := middleware.SetupMiddlewares(ctx, cfg.Model, middleware.Config{})
		if err != nil {
			return nil, fmt.Errorf("setup default middlewares: %w", err)
		}
		cfg.Middlewares = defaultMW
	}

	// 构建 ToolsConfig
	var toolsConfig adk.ToolsConfig
	if len(cfg.Tools) > 0 {
		toolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: cfg.Tools,
			},
		}
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Instruction:   cfg.Instruction,
		MaxIterations: cfg.MaxIterations,
		Model:         cfg.Model,
		ToolsConfig:   toolsConfig,
		Handlers:      cfg.Middlewares,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agent, nil
}
```

**注意:** `adk.BaseTool` 和 `tool.BaseTool` 需要确认类型一致，检查 eino 的类型定义。如果 `adk` 包没有导出 `BaseTool`，需要从 `components/tool` 包导入。实际写代码时需验证：

```go
// 检查 adk 包是否有 BaseTool 类型
```

**Step 2: 编译检查**

Run: `cd /home/lsk/projects/aimc-pro/aimc-go && go build ./assistant/...`

Expected: 可能编译失败（类型不匹配），根据错误调整 `cfg.Tools` 的类型。

**Step 3: Commit**

```bash
git add assistant/agent/agent.go
git commit -m "refactor(agent): inject model/tools/middlewares via AgentConfig, remove internal dependency creation"
```

---

## Task 5: 重构 `runner.go` — RunnerConfig 注入 store/sink

**Files:**
- Modify: `assistant/agent/runner.go`

**Step 1: 重写 runner.go，使用 RunnerOption 模式**

```go
package agent

import (
	"aimc-go/assistant/sink"
	"aimc-go/assistant/store"
	"context"
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

// RunnerOption Runner 配置选项
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

func NewRunner(agent adk.Agent, opts ...RunnerOption) *Runner {
	r := &Runner{
		inner: adk.NewRunner(context.Background(), adk.RunnerConfig{
			Agent:           agent,
			EnableStreaming: true,
		}),
		handler: &EventHandler{},
		// 默认值
		store: &store.JSONLStore{
			Dir:   "./data/sessions",
			Cache: make(map[string]*store.Session),
		},
		sink: &sink.StdoutSink{},
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
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
		Sink:      r.sink, // 使用 Runner 自己的 sink，而非硬编码
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

Run: `cd /home/lsk/projects/aimc-pro/aimc-go && go build ./assistant/...`

Expected: 可能需要调整 `assistant.go` 的调用方式。

**Step 3: Commit**

```bash
git add assistant/agent/runner.go
git commit -m "refactor(runner): inject store/sink via functional options, remove hardcoded defaults from processEventStream"
```

---

## Task 6: 更新 `assistant.go` — 适配新 API

**Files:**
- Modify: `assistant/assistant.go`

**Step 1: 重写 assistant.go，展示新 API 的使用方式**

```go
package assistant

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/agent/llm"
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

	// 1. 创建 model（调用方决定配置）
	cm, err := llm.NewChatModel(ctx, llm.DefaultConfig())
	if err != nil {
		panic(err)
	}

	// 2. 创建 agent（依赖注入）
	projectRoot := "/home/lsk/projects/eino-demo"
	ag, err := agent.New(ctx, agent.AgentConfig{
		Name:          "enio-assistant",
		Description:   "enio tutorial assistant",
		Instruction:   fmt.Sprintf(prompts.EinoTutorial, projectRoot, projectRoot, projectRoot, projectRoot),
		Model:         cm,
		MaxIterations: 30,
	})
	if err != nil {
		panic(err)
	}

	// 3. 创建 runner（可选注入 store/sink，不传则用默认值）
	runner := agent.NewRunner(ag)

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

**Step 2: 全量编译验证**

Run: `cd /home/lsk/projects/aimc-pro/aimc-go && go build ./assistant/...`

Expected: PASS

**Step 3: Commit**

```bash
git add assistant/assistant.go
git commit -m "refactor(assistant): adapt to new dependency injection API"
```

---

## Task 7: 最终编译 + 清理检查

**Files:**
- Verify: 整个 `assistant/` 包

**Step 1: 全量编译**

Run: `cd /home/lsk/projects/aimc-pro/aimc-go && go build ./assistant/...`

**Step 2: 检查是否有残留硬编码**

Run: `cd /home/lsk/projects/aimc-pro/aimc-go && grep -rn "sk-sp-\|/home/lsk/" assistant/agent/`

Expected: 只在 `defaults.go` 或注释中出现（如果有默认值便捷函数的话），核心逻辑中不应有。

**Step 3: 如果有残留，修复后提交**

```bash
git add -A
git commit -m "refactor: final cleanup of hardcoded values"
```

---

## 最终目录结构

```
assistant/
  assistant.go              # 调用方：展示 DI 用法
  agent/
    agent.go                # AgentConfig + New() — 接受依赖注入
    runner.go               # Runner + RunnerOption — 注入 store/sink
    runner_event.go         # EventHandler（不变）
    llm/provider.go         # Config struct — 无硬编码
    tools/tools.go          # InitTools(cm) — 接受 model
    tools/rag.go            # 不变
    tools/search.go         # 不变
    middleware/infra.go     # Config{SkillDir} — 路径参数化
    middleware/toolsafe.go  # 不变
    prompts/template.go     # 不变
  sink/sink.go              # 不变
  store/store.go            # 不变
```

## 重构前后对比

```go
// BEFORE: 硬编码一切
ag, err := agent.New(agent.Config{
    Name: "assistant",
    // model? tools? middleware? 全部内部创建
})

// AFTER: 调用方控制
cm, _ := llm.NewChatModel(ctx, llm.DefaultConfig())
ag, _ := agent.New(ctx, agent.AgentConfig{
    Name:    "assistant",
    Model:   cm,                              // 注入 model
    Tools:   myTools,                         // 可选：注入 tools
    Middlewares: myMWs,                       // 可选：注入 middlewares
})
runner := agent.NewRunner(ag,
    agent.WithStore(myStore),                 // 可选：注入 store
    agent.WithSink(mySink),                   // 可选：注入 sink
)
```
