package sink

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

// StdoutSink 标准输出
type StdoutSink struct {
	mu sync.Mutex
	w  *bufio.Writer
}

func NewStdoutSink() Sink {
	return &StdoutSink{
		w: bufio.NewWriterSize(os.Stdout, 1024),
	}
}

func (s *StdoutSink) Emit(c Chunk) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = fmt.Fprint(s.w, c.Content)
	_ = s.w.Flush()
}
