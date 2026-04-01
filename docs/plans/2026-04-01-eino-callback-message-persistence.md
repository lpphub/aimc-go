# 基于 eino Callback 重构消息持久化

## 现状问题

当前 `assistant/agent/runner.go` 中的消息持久化存在以下问题：

1. **消息持久化不完整** — 只在 `Run` 前后手动 `Append` user/assistant 消息，中间的 tool call / tool result 未持久化
2. **职责混杂** — `EventHandler` 同时负责 sink 输出和内容收集（Collector），违反单一职责
3. **未利用 eino 回调机制** — eino v0.8.5 提供了完整的 `callbacks.Handler` 系统，ADK Runner 原生支持 `WithCallbacks()`

## eino Callback 机制

### 核心 API

```go
import (
    "github.com/cloudwego/eino/callbacks"
    "github.com/cloudwego/eino/adk"
)

// Handler 接口
type Handler interface {
    OnStart(ctx, info *RunInfo, input CallbackInput) context.Context
    OnEnd(ctx, info *RunInfo, output CallbackOutput) context.Context
    OnError(ctx, info *RunInfo, err error) context.Context
    OnStartWithStreamInput(ctx, info *RunInfo, input *schema.StreamReader[CallbackInput]) context.Context
    OnEndWithStreamOutput(ctx, info *RunInfo, output *schema.StreamReader[CallbackOutput]) context.Context
}

// ADK 专用类型
type AgentCallbackInput struct {
    Input      *AgentInput  // 新运行时非 nil
    ResumeInfo *ResumeInfo  // 恢复运行时非 nil
}

type AgentCallbackOutput struct {
    Events *AsyncIterator[*AgentEvent]  // 事件流，每个 handler 独立 copy
}
```

### 注入方式

```go
// 单次运行注入
iter := runner.Run(ctx, messages, adk.WithCallbacks(handler))

// 指定特定 agent
iter := runner.Run(ctx, messages, adk.WithCallbacks(handler).DesignateAgent("MyAgent"))

// 全局注入（所有 graph/component 生效）
callbacks.AppendGlobalHandlers(handler)
```

## 推荐方案：三层架构

| 层 | 职责 | 实现方式 |
|---|---|---|
| **Store** | 消息持久化（读写 session history） | 保持现有 `store.Store` 接口 |
| **MessageCallback** | 生命周期钩子，在 OnStart/OnEnd 中调用 Store | 新增 callback handler |
| **EventHandler + Sink** | 实时输出（SSE / Stdout） | 保持不变，只负责输出 |

### 架构图

```
                    ┌─────────────────┐
                    │   adk.Runner    │
                    │   .Run(ctx,…)   │
                    └────────┬────────┘
                             │
              WithCallbacks(┌┴──────────────┐)
                            │MessageCallback│
                            └──┬─────────┬──┘
                       OnStart │         │ OnEnd
                               │         │
                    ┌──────────▼──┐  ┌───▼────────────┐
                    │ store.Append │  │ store.Append   │
                    │ (user msg)   │  │ (assistant msg │
                    └──────────────┘  │  tool msg)     │
                                      └────────────────┘

                    ┌──────────────────────────┐
                    │     EventHandler          │
                    │  (只负责 Sink 输出)        │
                    └──────────────────────────┘
```

## 具体代码

### 1. 新增 `assistant/agent/message_callback.go`

```go
package agent

import (
    "strings"

    "aimc-go/assistant/store"

    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/callbacks"
    "github.com/cloudwego/eino/schema"
    "context"
)

// NewMessageCallback 创建一个用于消息持久化的 callback handler。
// 在 OnStart 中持久化 user message，在 OnEnd 中异步消费事件流并持久化 assistant/tool 消息。
func NewMessageCallback(store store.Store, sessionID string) callbacks.Handler {
    return callbacks.NewHandlerBuilder().
        OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
            if info.Component != adk.ComponentOfAgent {
                return ctx
            }

            agentInput := adk.ConvAgentCallbackInput(input)
            if agentInput == nil || agentInput.Input == nil {
                return ctx
            }

            // 持久化 input 中的最后一条 user message
            msgs := agentInput.Input.Messages
            if len(msgs) > 0 {
                last := msgs[len(msgs)-1]
                if last.Role == schema.User {
                    _ = store.Append(ctx, sessionID, last)
                }
            }

            return ctx
        }).
        OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
            if info.Component != adk.ComponentOfAgent {
                return ctx
            }

            agentOutput := adk.ConvAgentCallbackOutput(output)
            if agentOutput == nil || agentOutput.Events == nil {
                return ctx
            }

            // 异步消费事件流，避免阻塞 agent 执行
            events := agentOutput.Events
            go func() {
                defer events.Close()

                var assistantContent strings.Builder
                var toolCalls []schema.ToolCall

                for {
                    event, ok := events.Next()
                    if !ok {
                        break
                    }

                    if event.Output == nil || event.Output.MessageOutput == nil {
                        continue
                    }

                    mv := event.Output.MessageOutput
                    msg, err := mv.GetMessage()
                    if err != nil {
                        continue
                    }

                    switch mv.Role {
                    case schema.Assistant:
                        assistantContent.WriteString(msg.Content)
                        toolCalls = append(toolCalls, msg.ToolCalls...)
                    case schema.Tool:
                        // 持久化 tool result
                        _ = store.Append(ctx, sessionID, msg)
                    }
                }

                // 持久化 assistant 消息
                if assistantContent.Len() > 0 {
                    assistantMsg := schema.AssistantMessage(assistantContent.String(), toolCalls)
                    _ = store.Append(ctx, sessionID, assistantMsg)
                }
            }()

            return ctx
        }).
        Build()
}
```

### 2. 简化 `assistant/agent/runner.go`

```go
package agent

import (
    "aimc-go/assistant/approval"
    "aimc-go/assistant/sink"
    "aimc-go/assistant/store"
    "context"
    "fmt"
    "strings"

    adkstore "github.com/cloudwego/eino-examples/adk/common/store"
    "github.com/cloudwego/eino/adk"
    "github.com/google/uuid"
)

type Runner struct {
    inner    *adk.Runner
    handler  *EventHandler
    store    store.Store
    sink     sink.Sink
    approver approval.ApprovalHandler
}

type RunnerOption func(*Runner)

func WithStore(s store.Store) RunnerOption {
    return func(r *Runner) { r.store = s }
}

func WithSink(s sink.Sink) RunnerOption {
    return func(r *Runner) { r.sink = s }
}

func WithApprovalHandler(p approval.ApprovalHandler) RunnerOption {
    return func(r *Runner) { r.approver = p }
}

func NewRunner(agent adk.Agent, opts ...RunnerOption) (*Runner, error) {
    r := &Runner{
        inner: adk.NewRunner(context.Background(), adk.RunnerConfig{
            Agent:           agent,
            EnableStreaming: true,
            CheckPointStore: adkstore.NewInMemoryStore(),
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

func (r *Runner) Run(ctx context.Context, sessionID, query string) error {
    if sessionID == "" {
        sessionID = uuid.New().String()
    }

    session, _ := r.store.GetOrCreate(ctx, sessionID)

    // 消息持久化由 callback 处理，不再手动 Append
    msgCb := NewMessageCallback(r.store, sessionID)

    iter := r.inner.Run(ctx, session.Messages,
        adk.WithCheckPointID(sessionID),
        adk.WithCallbacks(msgCb),
    )

    content, interruptInfo, err := r.processEventStream(ctx, iter)
    if err != nil {
        return err
    }

    for interruptInfo != nil {
        content, interruptInfo, err = r.handleInterrupt(ctx, sessionID, interruptInfo)
        if err != nil {
            return err
        }
    }

    // 不再需要手动 Append，由 MessageCallback.OnEnd 异步处理
    _ = content

    return nil
}

// processEventStream 保持不变，只负责 Sink 输出
func (r *Runner) processEventStream(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent]) (string, *adk.InterruptInfo, error) {
    ec := &EventContext{
        Ctx:       ctx,
        Collector: &strings.Builder{},
        Sink:      r.sink,
    }

    ec.Sink.Output(sink.Event{Type: "message", Content: "🤖: "})
    for {
        event, ok := iter.Next()
        if !ok {
            break
        }

        interruptInfo, err := r.handler.HandleEvent(ec, event)
        if err != nil {
            ec.Sink.Output(sink.Event{Type: "message", Content: event.Err.Error()})
            return "", nil, err
        }
        if interruptInfo != nil {
            return ec.Collector.String(), interruptInfo, nil
        }
    }

    return ec.Collector.String(), nil, nil
}

// Resume 和 handleInterrupt 保持不变
func (r *Runner) Resume(ctx context.Context, checkPointID string, resumeData map[string]any) (string, *adk.InterruptInfo, error) {
    msgCb := NewMessageCallback(r.store, checkPointID)

    events, err := r.inner.ResumeWithParams(ctx, checkPointID, &adk.ResumeParams{
        Targets: resumeData,
    })
    if err != nil {
        return "", nil, fmt.Errorf("failed to resume: %w", err)
    }
    // resume 时也要注册 callback
    _ = msgCb // callback 已通过 RunnerConfig 或需在 Resume 中注入
    return r.processEventStream(ctx, events)
}

func (r *Runner) handleInterrupt(ctx context.Context, checkPointID string, interruptInfo *adk.InterruptInfo) (string, *adk.InterruptInfo, error) {
    for _, ic := range interruptInfo.InterruptContexts {
        if !ic.IsRootCause {
            continue
        }

        if r.approver == nil {
            return "", nil, fmt.Errorf("interrupt occurred but no approval handler configured")
        }

        result, err := r.approver.GetApproval(ctx, ic)
        if err != nil {
            return "", nil, fmt.Errorf("approval failed: %w", err)
        }

        content, newInterruptInfo, err := r.Resume(ctx, checkPointID, map[string]any{
            ic.ID: result,
        })
        if err != nil {
            return "", nil, err
        }

        return content, newInterruptInfo, nil
    }

    return "", nil, fmt.Errorf("no root cause interrupt context found")
}
```

### 3. 可选：扩展 EventHandler，移除 Collector

如果希望进一步简化，可以让 `EventHandler` 不再收集内容（Collector 交给 callback 处理）：

```go
type EventContext struct {
    Ctx  context.Context
    Sink sink.Sink // 只保留 Sink，移除 Collector
}
```

但保留 Collector 也有好处：`processEventStream` 可以同步拿到最终内容用于 interrupt 判断，所以建议暂保留。

## 优点

1. **消息持久化更完整** — tool call / tool result 也会被持久化到 session history
2. **解耦** — Store 写入逻辑从 Runner 主流程抽离，可独立测试
3. **可扩展** — 未来加 tracing、metrics 只需再注册一个 callback handler：
   ```go
   iter := r.inner.Run(ctx, messages,
       adk.WithCallbacks(msgCb, tracingCb, metricsCb),
   )
   ```
4. **符合 eino 设计理念** — callback 就是为 cross-cutting concern 设计的

## 注意事项

1. **Events 流必须异步消费** — `OnEnd` 中的 `AgentCallbackOutput.Events` 需要 `go func()` 消费，否则会阻塞 agent 执行
2. **Resume 场景** — callback 的 `OnStart` 会收到 `ResumeInfo` 而非 `Input`，需要判断跳过重复持久化
3. **并发安全** — JSONLStore 的 `Append` 已有 mutex 保护，但注意 `go func()` 中的 context 可能已取消，生产环境建议用独立 context
4. **`Resume` 方法的 callback 注入** — 当前 `ResumeWithParams` 可能不支持 `WithCallbacks`，需确认 eino API 是否支持 resume 时的 callback 注入；如不支持，需在 resume 前通过全局 handler 注册

## 迁移步骤

1. 新增 `assistant/agent/message_callback.go`，实现 `NewMessageCallback`
2. 修改 `runner.go` 的 `Run` 方法，注入 callback 并移除手动 `Append`
3. 验证 tool call / tool result 被正确持久化
4. （可选）简化 `EventHandler`，移除不需要的字段
5. 测试 interrupt / resume 场景下的消息完整性
