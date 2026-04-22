package session

// ChunkType 输出片段类型
type ChunkType string

const (
	TypeAssistant   ChunkType = "assistant"
	TypeToolCall    ChunkType = "tool_call"
	TypeToolResult  ChunkType = "tool_result"
	TypeMessage     ChunkType = "message"
	TypeApproval    ChunkType = "approval"        // 审批请求
	TypeApprovalRes ChunkType = "approval_result" // 审批结果
	TypeError       ChunkType = "error"           // 错误信息
	TypeDone        ChunkType = "done"            // 对话结束信号
)

// Chunk 输出片段
type Chunk struct {
	Type    ChunkType      `json:"type"`
	Content string         `json:"content"`
	Meta    map[string]any `json:"meta,omitempty"`
}