package assistant

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/approval"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

func Run() {
	ctx := context.Background()

	assistantAgent, err := agent.New(ctx)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	sink := agent.StdoutSink()
	store := agent.JSONLStore("./data/sessions")
	approvalHandler := approval.NewCLIApprovalHandler(scanner, sink)

	runner, err := agent.NewRunner(assistantAgent,
		agent.WithStore(store),
		agent.WithSink(sink),
		agent.WithApprovalHandler(approvalHandler),
	)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	sessionID := "e69dfa6e-820a-4fcf-8a23-40107b0a324f"

	for {
		_, _ = fmt.Fprint(os.Stdout, "👤: ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		// 发送给 agent
		err = runner.Run(ctx, sessionID, line)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err = scanner.Err(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
