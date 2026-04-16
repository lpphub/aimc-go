package server

import (
	"aimc-go/assistant/session"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

// RunCLI 运行 CLI 模式
func RunCLI() {
	rt, err := NewRuntime()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)
	sessionID := uuid.New().String()

	sess := session.NewCLI(sessionID, session.NewStdoutWriter(), scanner)

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
	}
}