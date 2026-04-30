package session

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"aimc-go/assistant/types"
)

// Transport 传输层抽象（CLI、SSE、WebSocket 等）
type Transport interface {
	Emit(Event) error
	WaitInput(ctx context.Context) (InputEvent, error)
	Close()
}

// InputSink 支持外部向 transport 注入输入事件
type InputSink interface {
	Accept(ctx context.Context, ev InputEvent) error
}

type CLITransport struct {
	scanner *bufio.Scanner
}

func NewCLITransport(scanner *bufio.Scanner) *CLITransport {
	return &CLITransport{scanner: scanner}
}

func (t *CLITransport) Emit(e Event) error {
	switch e.Type {
	case TypeReasoning:
		_, err := fmt.Print("\033[90m" + e.Content + "\033[0m")
		return err
	default:
		_, err := fmt.Print(e.Content)
		return err
	}
}

func (t *CLITransport) WaitInput(ctx context.Context) (InputEvent, error) {
	if !t.scanner.Scan() {
		return InputEvent{}, t.scanner.Err()
	}
	response := strings.TrimSpace(t.scanner.Text())
	return InputEvent{
		Type: InputApproval,
		Data: &types.ApprovalResult{Approved: response == "y" || response == "yes"},
	}, nil
}

func (t *CLITransport) Close() {}

type SSETransport struct {
	w         http.ResponseWriter
	flusher   http.Flusher
	ctx       context.Context
	ch        chan InputEvent
	closed    chan struct{}
	closeOnce sync.Once
}

func NewSSETransport(ctx context.Context, w http.ResponseWriter, flusher http.Flusher) *SSETransport {
	return &SSETransport{
		w:       w,
		flusher: flusher,
		ctx:     ctx,
		ch:      make(chan InputEvent, 1),
		closed:  make(chan struct{}),
	}
}

func (t *SSETransport) Emit(e Event) error {
	if t.ctx != nil && t.ctx.Err() != nil {
		return t.ctx.Err()
	}

	// SSE 前端不展示 tool_result
	if e.Type == TypeToolResult {
		return nil
	}

	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	if _, err = fmt.Fprintf(t.w, "data: %s\n\n", data); err != nil {
		return err
	}

	if t.flusher != nil {
		t.flusher.Flush()
	}
	return nil
}

func (t *SSETransport) WaitInput(ctx context.Context) (InputEvent, error) {
	select {
	case ev := <-t.ch:
		return ev, nil
	case <-t.closed:
		return InputEvent{}, fmt.Errorf("transport closed")
	case <-ctx.Done():
		return InputEvent{}, ctx.Err()
	}
}

func (t *SSETransport) Accept(ctx context.Context, ev InputEvent) error {
	select {
	case t.ch <- ev:
		return nil
	case <-t.closed:
		return fmt.Errorf("transport closed")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *SSETransport) Close() {
	t.closeOnce.Do(func() {
		close(t.closed)
	})
}

type MultiTransport struct {
	Transports []Transport
}

func NewMultiTransport(ts ...Transport) *MultiTransport {
	return &MultiTransport{Transports: ts}
}

func (m *MultiTransport) Emit(e Event) error {
	for _, t := range m.Transports {
		if err := t.Emit(e); err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiTransport) WaitInput(ctx context.Context) (InputEvent, error) {
	return m.Transports[0].WaitInput(ctx)
}

func (m *MultiTransport) Close() {
	for _, t := range m.Transports {
		t.Close()
	}
}
