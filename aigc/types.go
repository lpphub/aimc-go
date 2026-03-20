package aigc

type ContentType string

const (
	ContentText  ContentType = "text"
	ContentImage ContentType = "image"
	ContentVideo ContentType = "video"
)

type ModelID string

type GenerateRequest struct {
	Model ModelID
	Type  ContentType

	Prompt string
	Params map[string]any
}

type GenerateResponse struct {
	Text string
	URL  string

	Meta map[string]any
}
