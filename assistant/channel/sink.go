package channel

// ChunkType 输出片段类型
type ChunkType string

const (
	TypeAssistant   ChunkType = "assistant"
	TypeToolCall    ChunkType = "tool_call"
	TypeToolResult  ChunkType = "tool_result"
	TypeMessage     ChunkType = "message"
	TypeApproval    ChunkType = "approval"         // 审批请求
	TypeApprovalRes ChunkType = "approval_result"  // 审批结果
	TypeError       ChunkType = "error"            // 错误信息
	TypeDone        ChunkType = "done"             // 对话结束信号
)

// Chunk 输出片段
type Chunk struct {
	Type    ChunkType
	Content string
	Meta    map[string]any
}

// Sink 实时输出展示接口
type Sink interface {
	Emit(Chunk)
}

// MultiSink 多 Sink 组合
type MultiSink struct {
	Sinks []Sink
}

func NewMultiSink(sinks ...Sink) *MultiSink {
	return &MultiSink{Sinks: sinks}
}

func (m *MultiSink) Emit(c Chunk) {
	for _, s := range m.Sinks {
		s.Emit(c)
	}
}