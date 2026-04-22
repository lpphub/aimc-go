package server

import (
	"aimc-go/assistant/approval"
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

	sess := NewCLI(sessionID, session.NewStdoutWriter(), scanner)

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

// NewCLI 创建 CLI 场景的 Session
func NewCLI(sessionID string, writer session.Writer, scanner *bufio.Scanner) *session.Session {
	sess := session.New(sessionID, writer)

	sess.OnInput = func(ctx context.Context) (*session.InputEvent, error) {
		if !scanner.Scan() {
			return nil, scanner.Err()
		}
		response := strings.TrimSpace(scanner.Text())
		approved := response == "y" || response == "yes"
		return &session.InputEvent{
			Type: session.InputApproval,
			Data: &approval.Result{Approved: approved},
		}, nil
	}

	return sess
}