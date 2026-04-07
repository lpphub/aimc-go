package channel

import (
	"aimc-go/assistant/approval"
	"bufio"
	"context"
	"fmt"
	"strings"
)

// NewCLIChannel 创建 CLI 场景的 Channel
func NewCLIChannel(sessionID string, scanner *bufio.Scanner) *Channel {
	ch := NewChannel(sessionID, NewStdoutWriter())
	
	ch.OnInput = func(ctx context.Context) (*InputEvent, error) {
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

	return ch
}
