package session

import (
	"bufio"
	"context"
	"strings"

	"aimc-go/assistant/approval"
)

// NewCLI 创建 CLI 场景的 Session
func NewCLI(sessionID string, writer Writer, scanner *bufio.Scanner) *Session {
	sess := New(sessionID, writer)

	sess.OnInput = func(ctx context.Context) (*InputEvent, error) {
		if !scanner.Scan() {
			return nil, scanner.Err()
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
