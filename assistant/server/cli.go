package server

import (
	"aimc-go/assistant/agent/callback"
	"aimc-go/assistant/session"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/callbacks"
	"github.com/google/uuid"
)

// RunCLI 运行 CLI 模式
func RunCLI() {
	rt, err := NewRuntime()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	stats := &callback.UsageStats{}
	ctx := callback.WithUsageStats(context.Background(), stats)
	// 全局用量统计
	callbacks.AppendGlobalHandlers(callback.NewUsageHandler())

	scanner := bufio.NewScanner(os.Stdin)
	sessionID := uuid.New().String()

	sess := NewCLI(sessionID, session.NewStdoutSink(), scanner)

	for {
		fmt.Print("👤: ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		if err := rt.Run(ctx, sess, line); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		stats.Report()
	}
}

// NewCLI 创建 CLI 场景的 Session
func NewCLI(sessionID string, sink session.Sink, scanner *bufio.Scanner) *session.Session {
	sess := session.New(sessionID, sink, false)

	sess.OnInput = func(ctx context.Context) (*session.InputEvent, error) {
		if !scanner.Scan() {
			return nil, scanner.Err()
		}
		response := strings.TrimSpace(scanner.Text())
		approved := response == "y" || response == "yes"
		return &session.InputEvent{
			Type: session.InputApproval,
			Data: &session.ApprovalResult{Approved: approved},
		}, nil
	}

	return sess
}
