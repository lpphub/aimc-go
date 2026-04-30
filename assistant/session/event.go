package session

type EventType string

const (
	TypeAssistant  EventType = "assistant"   // 助手回复
	TypeReasoning  EventType = "reasoning"   // 思考过程
	TypeToolCall   EventType = "tool_call"   // 工具调用
	TypeToolResult EventType = "tool_result" // 工具结果
	TypeApproval   EventType = "approval"    // 审批请求
	TypeMessage    EventType = "message"     // 普通消息
)

type Event struct {
	Type    EventType      `json:"type"`
	Content string         `json:"content"`
	Meta    map[string]any `json:"meta,omitempty"`
}

type InputType string

const (
	InputApproval InputType = "approval"
	InputFollowup InputType = "followup"
)

type InputEvent struct {
	Type InputType
	Data any
}
