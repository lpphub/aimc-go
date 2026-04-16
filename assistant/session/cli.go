package session

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"aimc-go/assistant/approval"
)

// NewCLI 创建 CLI 场景的 Session
func NewCLI(sessionID string, scanner *bufio.Scanner) *Session {
	sess := New(sessionID, NewStdoutWriter())

	sess.OnInput = func(ctx context.Context) (*InputEvent, error) {
		if !scanner.Scan() {
			return nil, fmt.Errorf("failed to read input: %w", scanner.Err())
		}
		response := strings.TrimSpace(scanner.Text())

		approved := response == "y" || response == "yes"
		return &InputEvent{
			Type: InputApproval,
			Data: &approval.Result{Approved: approved},
		}, nil
	}

	return sess
}