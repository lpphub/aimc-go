package session

// InputType 输入事件类型
type InputType string

const (
	InputApproval    InputType = "approval"
	InputUserMessage InputType = "user_message"
)

// InputEvent 输入事件
type InputEvent struct {
	Type InputType
	Data any
}