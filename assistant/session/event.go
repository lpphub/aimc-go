package session

// EventType 事件类型
type EventType string

const (
	TypeAssistant   EventType = "assistant"
	TypeToolCall    EventType = "tool_call"
	TypeToolResult  EventType = "tool_result"
	TypeMessage     EventType = "message"
	TypeApproval    EventType = "approval"        // 审批请求
	TypeApprovalRes EventType = "approval_result" // 审批结果
	TypeError       EventType = "error"           // 错误信息
	TypeDone        EventType = "done"            // 对话结束信号
)

// Event 输出事件
type Event struct {
	Type    EventType      `json:"type"`
	Content string         `json:"content"`
	Meta    map[string]any `json:"meta,omitempty"`
}