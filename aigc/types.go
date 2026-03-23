package aigc

type TaskType string

const (
	TaskMarketingCopy  TaskType = "marketing_copy"
	TaskMarketingImage TaskType = "marketing_image"
	TaskGeneralText    TaskType = "general_text"
)

type ModelID string

type GenerateRequest struct {
	Task   TaskType
	Model  ModelID
	Prompt string
	Params map[string]any
}

type GenerateResponse struct {
	Text string
	URL  string
	Meta map[string]any
}
