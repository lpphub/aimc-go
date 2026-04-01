package sink

import (
	"fmt"
)

type Event struct {
	Type    string // assistant | tool_call | tool_result | log
	Content string
}

// Sink 事件消息输出层
type Sink interface {
	Output(e Event)
}

type MultiSink struct {
	Sinks []Sink
}

func (m *MultiSink) Output(e Event) {
	for _, s := range m.Sinks {
		s.Output(e)
	}
}

type StdoutSink struct{}

func (s *StdoutSink) Output(e Event) {
	fmt.Print(e.Content)
}

type SSESink struct {
}

func (s *SSESink) Output(e Event) {
	//todo
}
