package server

import (
	"aimc-go/assistant/agent/callbacks"
	"aimc-go/assistant/session"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

func RunCLI() {
	rt, err := NewRuntime()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cbs := callbacks.Init()
	ctx := callbacks.WithUsageStats(context.Background(), cbs.UsageStats)

	scanner := bufio.NewScanner(os.Stdin)
	sessionID := uuid.New().String()

	sess := session.New(sessionID, session.NewCLIEndpoint(scanner))

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

		cbs.UsageStats.Report()
	}
}
