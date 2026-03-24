# AIGC Module Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign the `aigc` module with layered architecture supporting OpenAI/Gemini providers, task-based routing, and prompt management.

**Architecture:** Layered design with Router, Registry, Model interface, and prompts package. Each provider implements Model interface using go-openai/go-genai SDKs.

**Tech Stack:** Go, github.com/sashabaranov/go-openai, google.golang.org/genai

---

## Dependencies

First, add the required Go dependencies:

```bash
go get github.com/sashabaranov/go-openai
go get google.golang.org/genai
```

---

### Task 1: Core Types

**Files:**
- Rewrite: `aigc/types.go`

**Step 1: Rewrite types.go with new types**

```go
package aigc

type TaskType string

const (
    TaskMarketingCopy  TaskType = "marketing_copy"
    TaskMarketingImage TaskType = "marketing_image"
    TaskGeneralText    TaskType = "general_text"
)

type ModelID string

type GenerateRequest struct {
    Task   TaskType       // Task type for routing
    Model  ModelID        // Optional: override default model
    Prompt string         // Prompt content
    Params map[string]any // Extra params
}

type GenerateResponse struct {
    Text string
    URL  string         // Image URL
    Meta map[string]any // Token usage, etc.
}
```

**Step 2: Verify compilation**

```bash
go build ./aigc/...
```

**Step 3: Commit**

```bash
 git add aigc/types.go && git commit -m "refactor(aigc): add TaskType and update request/response types"
```

---

### Task 2: Model Interface

**Files:**
- Modify: `aigc/model.go`

**Step 1: Verify model.go stays unchanged (already correct)**

```go
package aigc

import "context"

type Model interface {
    ID() ModelID
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
}
```

No changes needed, Model interface is already clean.

**Step 2: Commit if needed**

```bash
 git commit --allow-empty -m "chore(aigc): verify model interface unchanged"
```

---

### Task 3: Registry

**Files:**
- Rewrite: `aigc/registry.go`

**Step 1: Update registry.go with List method**

```go
package aigc

import "fmt"

type Registry struct {
    models map[ModelID]Model
}

func NewRegistry() *Registry {
    return &Registry{
        models: map[ModelID]Model{},
    }
}

func (r *Registry) Register(m Model) {
    r.models[m.ID()] = m
}

func (r *Registry) Get(id ModelID) (Model, error) {
    m, ok := r.models[id]
    if !ok {
        return nil, fmt.Errorf("model not found: %s", id)
    }
    return m, nil
}

func (r *Registry) List() []ModelID {
    ids := make([]ModelID, 0, len(r.models))
    for id := range r.models {
        ids = append(ids, id)
    }
    return ids
}
```

**Step 2: Verify compilation**

```bash
 go build ./aigc/...
```

**Step 3: Commit**

```bash
 git add aigc/registry.go && git commit -m "feat(aigc): add List method to Registry"
```

---

### Task 4: Router

**Files:**
- Create: `aigc/router.go`

**Step 1: Create router.go**

```go
package aigc

type Router struct {
    routes map[TaskType]ModelID
}

func NewRouter() *Router {
    return &Router{
        routes: make(map[TaskType]ModelID),
    }
}

func (r *Router) Register(task TaskType, model ModelID) {
    r.routes[task] = model
}

func (r *Router) Resolve(req *GenerateRequest) ModelID {
    if req.Model != "" {
        return req.Model
    }
    return r.routes[req.Task]
}
```

**Step 2: Write router test**

Create: `aigc/router_test.go`

```go
package aigc

import "testing"

func TestRouter_Resolve_ExplicitOverride(t *testing.T) {
    r := NewRouter()
    r.Register(TaskMarketingCopy, "openai-gpt4o")

    req := &GenerateRequest{
        Task:  TaskMarketingCopy,
        Model: "gemini-2.0-flash",
    }

    got := r.Resolve(req)
    if got != "gemini-2.0-flash" {
        t.Errorf("expected gemini-2.0-flash, got %s", got)
    }
}

func TestRouter_Resolve_Default(t *testing.T) {
    r := NewRouter()
    r.Register(TaskMarketingCopy, "openai-gpt4o")

    req := &GenerateRequest{Task: TaskMarketingCopy}

    got := r.Resolve(req)
    if got != "openai-gpt4o" {
        t.Errorf("expected openai-gpt4o, got %s", got)
    }
}

func TestRouter_Resolve_NoDefault(t *testing.T) {
    r := NewRouter()

    req := &GenerateRequest{Task: TaskGeneralText}

    got := r.Resolve(req)
    if got != "" {
        t.Errorf("expected empty, got %s", got)
    }
}
```

**Step 3: Run tests**

```bash
 go test ./aigc/ -run TestRouter -v
```

Expected: all pass

**Step 4: Commit**

```bash
 git add aigc/router.go aigc/router_test.go && git commit -m "feat(aigc): add Router for task-based model routing"
```

---

### Task 5: Prompt Templates

**Files:**
- Create: `aigc/prompts/marketing.go`

**Step 1: Create prompts directory and marketing.go**

```go
package prompts

import "aimc-go/aigc"

var Templates = map[aigc.TaskType]string{
    aigc.TaskMarketingCopy:  MarketingCopyTemplate,
    aigc.TaskMarketingImage: MarketingImageTemplate,
}

const MarketingCopyTemplate = `你是一位资深营销文案专家。请根据以下需求撰写营销文案：

需求：%s

要求：
- 语言简洁有力
- 突出卖点
- 适合社交媒体传播`

const MarketingImageTemplate = `Generate a professional marketing image for: %s

Style: modern, clean, professional, eye-catching
Aspect ratio: 16:9
Color scheme: vibrant and brand-appropriate`
```

**Step 2: Verify compilation**

```bash
 go build ./aigc/prompts/...
```

**Step 3: Commit**

```bash
 git add aigc/prompts/ && git commit -m "feat(aigc): add prompt templates for marketing tasks"
```

---

### Task 6: Client Refactor

**Files:**
- Rewrite: `aigc/client.go`

**Step 1: Update client.go with router integration and prompt building**

```go
package aigc

import (
    "aimc-go/aigc/prompts"
    "context"
    "fmt"
)

type Client struct {
    registry *Registry
    router   *Router
}

func NewClient(reg *Registry, router *Router) *Client {
    return &Client{
        registry: reg,
        router:   router,
    }
}

func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
    modelID := c.router.Resolve(req)
    if modelID == "" {
        return nil, fmt.Errorf("no model resolved for task: %s", req.Task)
    }

    model, err := c.registry.Get(modelID)
    if err != nil {
        return nil, err
    }

    req.Prompt = c.buildPrompt(req)
    return model.Generate(ctx, req)
}

func (c *Client) MarketingCopy(ctx context.Context, input string) (*GenerateResponse, error) {
    return c.Generate(ctx, &GenerateRequest{
        Task:   TaskMarketingCopy,
        Prompt: input,
    })
}

func (c *Client) MarketingImage(ctx context.Context, input string) (*GenerateResponse, error) {
    return c.Generate(ctx, &GenerateRequest{
        Task:   TaskMarketingImage,
        Prompt: input,
    })
}

func (c *Client) buildPrompt(req *GenerateRequest) string {
    if tmpl, ok := prompts.Templates[req.Task]; ok {
        return fmt.Sprintf(tmpl, req.Prompt)
    }
    return req.Prompt
}
```

**Step 2: Write client test with mock model**

Create: `aigc/client_test.go`

```go
package aigc

import (
    "context"
    "testing"
)

type mockModel struct {
    id ModelID
}

func (m *mockModel) ID() ModelID { return m.id }

func (m *mockModel) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
    return &GenerateResponse{Text: "mock: " + req.Prompt}, nil
}

func TestClient_MarketingCopy(t *testing.T) {
    reg := NewRegistry()
    reg.Register(&mockModel{id: "mock-openai"})

    router := NewRouter()
    router.Register(TaskMarketingCopy, "mock-openai")

    client := NewClient(reg, router)

    resp, err := client.MarketingCopy(context.Background(), "推广运动鞋")
    if err != nil {
        t.Fatal(err)
    }

    if resp.Text == "" {
        t.Error("expected non-empty response")
    }

    if resp.Text != "mock: 推广运动鞋" {
        t.Errorf("unexpected response: %s", resp.Text)
    }
}

func TestClient_Generate_ModelOverride(t *testing.T) {
    reg := NewRegistry()
    reg.Register(&mockModel{id: "model-a"})
    reg.Register(&mockModel{id: "model-b"})

    router := NewRouter()
    router.Register(TaskMarketingCopy, "model-a")

    client := NewClient(reg, router)

    resp, err := client.Generate(context.Background(), &GenerateRequest{
        Task:   TaskMarketingCopy,
        Model:  "model-b",
        Prompt: "test",
    })
    if err != nil {
        t.Fatal(err)
    }
    if resp.Text != "mock: test" {
        t.Errorf("unexpected: %s", resp.Text)
    }
}

func TestClient_Generate_NoModel(t *testing.T) {
    reg := NewRegistry()
    router := NewRouter()

    client := NewClient(reg, router)

    _, err := client.Generate(context.Background(), &GenerateRequest{
        Task:   TaskGeneralText,
        Prompt: "test",
    })
    if err == nil {
        t.Error("expected error for unresolved model")
    }
}
```

**Step 3: Remove old client_test.go content (replaced above)**

**Step 4: Run tests**

```bash
 go test ./aigc/ -v
```

Expected: all pass

**Step 5: Commit**

```bash
 git add aigc/client.go aigc/client_test.go && git commit -m "refactor(aigc): update Client with router integration and prompt building"
```

---

### Task 7: OpenAI Provider

**Files:**
- Rewrite: `aigc/models/openai.go`

**Step 1: Rewrite openai.go with real SDK integration**

```go
package models

import (
    "aimc-go/aigc"
    "context"
    "fmt"

    openai "github.com/sashabaranov/go-openai"
)

type OpenAI struct {
    client *openai.Client
    model  string
}

func NewOpenAI(apiKey string) *OpenAI {
    return &OpenAI{
        client: openai.NewClient(apiKey),
        model:  "gpt-4o",
    }
}

func (m *OpenAI) ID() aigc.ModelID {
    return "openai-gpt4o"
}

func (m *OpenAI) Generate(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {
    switch req.Task {
    case aigc.TaskMarketingImage:
        return m.generateImage(ctx, req)
    default:
        return m.generateText(ctx, req)
    }
}

func (m *OpenAI) generateText(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {
    resp, err := m.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: m.model,
        Messages: []openai.ChatCompletionMessage{
            {Role: openai.ChatMessageRoleUser, Content: req.Prompt},
        },
    })
    if err != nil {
        return nil, fmt.Errorf("openai text generation failed: %w", err)
    }

    text := resp.Choices[0].Message.Content
    return &aigc.GenerateResponse{
        Text: text,
        Meta: map[string]any{
            "usage": resp.Usage,
        },
    }, nil
}

func (m *OpenAI) generateImage(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {
    size := openai.CreateImageSize1792x1024
    if s, ok := req.Params["size"].(string); ok {
        size = s
    }

    resp, err := m.client.CreateImage(ctx, openai.ImageRequest{
        Prompt: req.Prompt,
        Model:  openai.CreateImageModelDallE3,
        Size:   size,
        N:      1,
    })
    if err != nil {
        return nil, fmt.Errorf("openai image generation failed: %w", err)
    }

    return &aigc.GenerateResponse{
        URL: resp.Data[0].URL,
    }, nil
}
```

**Step 2: Verify compilation**

```bash
 go build ./aigc/models/...
```

**Step 3: Commit**

```bash
 git add aigc/models/openai.go && git commit -m "feat(aigc): implement OpenAI provider with go-openai SDK"
```

---

### Task 8: Gemini Provider

**Files:**
- Rewrite: `aigc/models/gemini.go`

**Step 1: Rewrite gemini.go with real SDK integration**

```go
package models

import (
    "aimc-go/aigc"
    "context"
    "fmt"

    "google.golang.org/genai"
)

type Gemini struct {
    client *genai.Client
    model  string
}

func NewGemini(ctx context.Context, apiKey string) (*Gemini, error) {
    client, err := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey:  apiKey,
        Backend: genai.BackendGeminiAPI,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create gemini client: %w", err)
    }

    return &Gemini{
        client: client,
        model:  "gemini-2.0-flash",
    }, nil
}

func (m *Gemini) ID() aigc.ModelID {
    return "gemini-2.0-flash"
}

func (m *Gemini) Generate(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {
    switch req.Task {
    case aigc.TaskMarketingImage:
        return m.generateImage(ctx, req)
    default:
        return m.generateText(ctx, req)
    }
}

func (m *Gemini) generateText(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {
    result, err := m.client.Models.GenerateContent(ctx, m.model, genai.Text(req.Prompt), nil)
    if err != nil {
        return nil, fmt.Errorf("gemini text generation failed: %w", err)
    }

    text := result.Text()
    return &aigc.GenerateResponse{
        Text: text,
        Meta: map[string]any{
            "usage": result.UsageMetadata,
        },
    }, nil
}

func (m *Gemini) generateImage(ctx context.Context, req *aigc.GenerateRequest) (*aigc.GenerateResponse, error) {
    result, err := m.client.Models.GenerateContent(ctx, "imagen-3.0-generate-002", genai.Text(req.Prompt), nil)
    if err != nil {
        return nil, fmt.Errorf("gemini image generation failed: %w", err)
    }

    if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
        part := result.Candidates[0].Content.Parts[0]
        if part.InlineData != nil {
            return &aigc.GenerateResponse{
                URL: fmt.Sprintf("data:%s;base64,...", part.InlineData.MIMEType),
                Meta: map[string]any{
                    "mime_type": part.InlineData.MIMEType,
                },
            }, nil
        }
    }

    return nil, fmt.Errorf("no image data in response")
}
```

**Step 2: Verify compilation**

```bash
 go build ./aigc/models/...
```

**Step 3: Commit**

```bash
 git add aigc/models/gemini.go && git commit -m "feat(aigc): implement Gemini provider with go-genai SDK"
```

---

### Task 9: Update Test

**Files:**
- Modify: `aigc/client_test.go`

**Step 1: Verify test still passes after provider changes**

```bash
 go test ./aigc/ -v
```

The test uses mockModel, so provider changes don't affect it.

**Step 2: Commit**

```bash
 git commit --allow-empty -m "test(aigc): verify client tests pass with new providers"
```

---

### Task 10: Integration Verification

**Step 1: Full build check**

```bash
 go build ./...
```

**Step 2: Run all tests**

```bash
 go test ./aigc/... -v
```

**Step 3: Verify no lint issues**

```bash
 go vet ./aigc/...
```

**Step 4: Final commit**

```bash
 git add -A && git commit -m "refactor(aigc): complete module redesign with layered architecture"
```

---

## File Summary

| File | Action | Purpose |
|------|--------|---------|
| `aigc/types.go` | Rewrite | Add TaskType, update request/response |
| `aigc/model.go` | Keep | Model interface (unchanged) |
| `aigc/registry.go` | Modify | Add List() method |
| `aigc/router.go` | Create | Task-based model routing |
| `aigc/router_test.go` | Create | Router unit tests |
| `aigc/client.go` | Rewrite | Integrate router + prompt building |
| `aigc/client_test.go` | Rewrite | Tests with mock model |
| `aigc/prompts/marketing.go` | Create | Prompt templates |
| `aigc/models/openai.go` | Rewrite | Real OpenAI SDK integration |
| `aigc/models/gemini.go` | Rewrite | Real Gemini SDK integration |
