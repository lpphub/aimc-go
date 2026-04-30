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

func RunCLI() {
	rt, err := NewRuntime()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 全局用量统计
	stats := &callback.UsageStats{}
	ctx := callback.WithUsageStats(context.Background(), stats)
	usageCallback := callback.NewUsageHandler()
	callbacks.AppendGlobalHandlers(usageCallback)

	scanner := bufio.NewScanner(os.Stdin)
	sessionID := uuid.New().String()

	sess := session.New(sessionID, session.NewCLITransport(scanner))

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
