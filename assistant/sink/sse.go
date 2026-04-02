package sink

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// SSESink SSE 推送 sink
type SSESink struct {
	mu      sync.Mutex
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSESink 创建 SSE sink
func NewSSESink(w http.ResponseWriter, flusher http.Flusher) Sink {
	return &SSESink{
		w:       w,
		flusher: flusher,
	}
}

func (s *SSESink) Emit(c Chunk) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(c)
	if err != nil {
		// marshal error, emit as error chunk
		data, _ = json.Marshal(Chunk{
			Type:    TypeError,
			Content: err.Error(),
		})
	}
	fmt.Fprintf(s.w, "data: %s\n\n", data)
	s.flusher.Flush()
}