package channel

import (
	"aimc-go/assistant/approval"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// StdoutSink 标准输出（带打字机效果）
type StdoutSink struct{}

func NewStdoutSink() Sink {
	return &StdoutSink{}
}

func (s *StdoutSink) Emit(c Chunk) {
	if c.Type == TypeAssistant {
		// 打字机效果：逐字符输出
		for _, r := range c.Content {
			fmt.Fprintf(os.Stdout, "%c", r)
			os.Stdout.Sync() // 确保立即输出
			time.Sleep(40 * time.Millisecond)
		}
	} else {
		fmt.Fprint(os.Stdout, c.Content)
	}
}

// NewCLIChannel 创建 CLI 场景的 Channel
func NewCLIChannel(sessionID string, scanner *bufio.Scanner) *Channel {
	ch := NewChannel(sessionID, NewStdoutSink())

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
