package runtime

import (
	"aimc-go/assistant/approval"
	"aimc-go/assistant/sink"
	"bufio"
	"context"
	"fmt"
	"strings"
)

// CLISessionBuilder CLI 场景的 SessionBuilder
type CLISessionBuilder struct {
	scanner *bufio.Scanner
}

// NewCLISessionBuilder 创建 CLI SessionBuilder
func NewCLISessionBuilder(scanner *bufio.Scanner) *CLISessionBuilder {
	if scanner == nil {
		panic("scanner cannot be nil")
	}
	return &CLISessionBuilder{scanner: scanner}
}

// Build 创建 CLI Session
func (b *CLISessionBuilder) Build(ctx context.Context, sessionID string) (*Session, error) {
	session := NewSession(ctx, sessionID, sink.NewStdoutSink())

	// 注册阻塞回调，直接读 stdin
	session.OnInput = func(ctx context.Context) (*InputEvent, error) {
		if !b.scanner.Scan() {
			return nil, fmt.Errorf("failed to read input: %w", b.scanner.Err())
		}
		response := strings.TrimSpace(b.scanner.Text())

		approved := response == "y" || response == "yes"
		return &InputEvent{
			Type: InputApproval,
			Data: &approval.ApprovalResult{Approved: approved},
		}, nil
	}

	return session, nil
}