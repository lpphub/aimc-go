# AIGC Module Redesign

## Overview

Redesign the `aigc` module to support dynamic model switching, prompt management, and extensible provider architecture for generating marketing copy and images.

## Requirements

- Support OpenAI and Gemini providers
- Generate text (marketing copy) and image content
- Task-based model routing with manual override
- Prompt templates as code constants
- Use `go-openai` and `go-genai` SDKs
- Clean package structure, easy to extend

## Architecture: Layered Design

```
aigc/
├── client.go          # Unified entry point
├── types.go           # Core types: TaskType, GenerateRequest, GenerateResponse
├── model.go           # Model interface
├── registry.go        # Model registration center
├── router.go          # TaskType → default ModelID routing
├── models/
│   ├── openai.go      # OpenAI implementation (go-openai SDK)
│   └── gemini.go      # Gemini implementation (go-genai SDK)
└── prompts/
    └── marketing.go   # Marketing prompt templates as constants
```

## Core Types

```go
type TaskType string
const (
    TaskMarketingCopy  TaskType = "marketing_copy"
    TaskMarketingImage TaskType = "marketing_image"
    TaskGeneralText    TaskType = "general_text"
)

type GenerateRequest struct {
    Task     TaskType        // Task type for routing
    Model    ModelID         // Optional: override default model
    Prompt   string          // Prompt content
    Params   map[string]any  // Extra params (temperature, size, etc.)
}

type GenerateResponse struct {
    Text string
    URL  string         // Image URL
    Meta map[string]any // Token usage, etc.
}
```

## Model Interface

```go
type Model interface {
    ID() ModelID
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
}
```

## Router

Maps TaskType to default ModelID. Explicit Model in request overrides default.

```go
type Router struct {
    routes map[TaskType]ModelID
}

func (r *Router) Register(task TaskType, model ModelID)
func (r *Router) Resolve(req *GenerateRequest) ModelID  // returns req.Model if set, else default
```

## Registry

Registers Model instances by ID.

```go
type Registry struct {
    models map[ModelID]Model
}

func (r *Registry) Register(m Model)
func (r *Registry) Get(id ModelID) (Model, error)
func (r *Registry) List() []ModelID
```

## Client

```go
type Client struct {
    registry *Registry
    router   *Router
}

func NewClient(reg *Registry, router *Router) *Client
func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
func (c *Client) MarketingCopy(ctx context.Context, input string) (*GenerateResponse, error)
func (c *Client) MarketingImage(ctx context.Context, input string) (*GenerateResponse, error)
```

Generate flow:
1. Resolve model via Router
2. Get Model instance from Registry
3. Build full prompt from template
4. Call Model.Generate()

## Provider Implementations

### OpenAI (models/openai.go)
- Uses `github.com/sashabaranov/go-openai`
- Text: ChatCompletion API
- Image: DALL-E API via ImageRequest

### Gemini (models/gemini.go)
- Uses `google.golang.org/genai`
- Text: `client.Models.GenerateContent()`
- Image: Imagen API

Each provider encapsulates SDK-specific logic, exposes only the Model interface.

## Prompt Templates (prompts/marketing.go)

```go
var Templates = map[aigc.TaskType]string{
    TaskMarketingCopy:  MarketingCopyTemplate,
    TaskMarketingImage: MarketingImageTemplate,
}

const MarketingCopyTemplate = `你是一位资深营销文案专家。请根据以下需求撰写营销文案：
需求：%s
要求：语言简洁有力、突出卖点、适合社交媒体传播`
```

## Usage Example

```go
reg := aigc.NewRegistry()
reg.Register(models.NewOpenAI(os.Getenv("OPENAI_API_KEY")))
reg.Register(models.NewGemini(ctx, os.Getenv("GEMINI_API_KEY")))

router := aigc.NewRouter()
router.Register(aigc.TaskMarketingCopy, "openai-gpt4o")
router.Register(aigc.TaskMarketingImage, "gemini-2.0-flash")

client := aigc.NewClient(reg, router)

// Auto-routed
resp, _ := client.MarketingCopy(ctx, "推广新款运动鞋")

// Manual override
resp, _ := client.Generate(ctx, &aigc.GenerateRequest{
    Task:   aigc.TaskMarketingCopy,
    Model:  "gemini-2.0-flash",
    Prompt: "推广新款运动鞋",
})
```

## Extension Points

- Add new provider: create `models/newprovider.go`, implement Model interface
- Add new task type: add to TaskType const, add prompt template, configure router default
- Add new content type: extend ContentType, add provider method
