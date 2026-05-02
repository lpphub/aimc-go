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

// Endpoint 交互端点抽象（CLI、SSE、WebSocket 等）
type Endpoint interface {
	Emit(Event) error
	WaitInput(ctx context.Context) (InputEvent, error)
	Close()
}

// InputSink 支持外部向 endpoint 注入输入事件（SSE、WebSocket ）
type InputSink interface {
	Accept(ctx context.Context, ev InputEvent) error
}

type CLIEndpoint struct {
	scanner *bufio.Scanner
}

func NewCLIEndpoint(scanner *bufio.Scanner) *CLIEndpoint {
	return &CLIEndpoint{scanner: scanner}
}

func (t *CLIEndpoint) Emit(e Event) error {
	switch e.Type {
	case TypeReasoning:
		_, err := fmt.Print("\033[90m" + e.Content + "\033[0m")
		return err
	default:
		_, err := fmt.Print(e.Content)
		return err
	}
}

func (t *CLIEndpoint) WaitInput(ctx context.Context) (InputEvent, error) {
	if !t.scanner.Scan() {
		return InputEvent{}, t.scanner.Err()
	}
	response := strings.TrimSpace(t.scanner.Text())
	return InputEvent{
		Type: InputApproval,
		Data: &types.ApprovalResult{Approved: response == "y" || response == "yes"},
	}, nil
}

func (t *CLIEndpoint) Close() {}

type SSEEndpoint struct {
	w         http.ResponseWriter
	flusher   http.Flusher
	ctx       context.Context
	ch        chan InputEvent
	closed    chan struct{}
	closeOnce sync.Once
}

func NewSSEEndpoint(ctx context.Context, w http.ResponseWriter, flusher http.Flusher) *SSEEndpoint {
	return &SSEEndpoint{
		w:       w,
		flusher: flusher,
		ctx:     ctx,
		ch:      make(chan InputEvent, 1),
		closed:  make(chan struct{}),
	}
}

func (t *SSEEndpoint) Emit(e Event) error {
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

func (t *SSEEndpoint) WaitInput(ctx context.Context) (InputEvent, error) {
	select {
	case ev := <-t.ch:
		return ev, nil
	case <-t.closed:
		return InputEvent{}, fmt.Errorf("endpoint closed")
	case <-ctx.Done():
		return InputEvent{}, ctx.Err()
	}
}

func (t *SSEEndpoint) Accept(ctx context.Context, ev InputEvent) error {
	select {
	case t.ch <- ev:
		return nil
	case <-t.closed:
		return fmt.Errorf("endpoint closed")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *SSEEndpoint) Close() {
	t.closeOnce.Do(func() {
		close(t.closed)
	})
}
