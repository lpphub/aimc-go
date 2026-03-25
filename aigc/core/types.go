package core

import "context"

type TaskType string

const (
	TaskMarketingCopy  TaskType = "marketing_copy"
	TaskMarketingImage TaskType = "marketing_image"
	TaskGeneralText    TaskType = "general_text"
)

type ModelID string

type Message struct {
	Role      string
	Content   string
	ToolCalls []ToolCall
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type GenerateRequest struct {
	Task     TaskType
	Model    ModelID
	Messages []Message
	Prompt   string
	Params   map[string]any
	Tools    []ToolDefinition
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type GenerateResponse struct {
	Text      string
	URL       string
	ToolCalls []ToolCall
	Meta      map[string]any
}

type ToolFunc func(ctx context.Context, args string) (string, error)
