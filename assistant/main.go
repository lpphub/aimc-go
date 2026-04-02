package assistant

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/runtime"
	"aimc-go/assistant/store"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

func Cli() {
	ctx := context.Background()

	assistantAgent, err := agent.New(ctx)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	builder := runtime.NewCLISessionBuilder(scanner)
	jsonlStore := store.NewJSONLStore("./data/sessions")

	rt, err := runtime.NewRuntime(assistantAgent, runtime.WithStore(jsonlStore))
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	sessionID := "e69dfa6e-820a-4fcf-8a23-40107b0a324f"

	// Session 代表一个会话，在多轮对话期间保持
	session, err := builder.Build(ctx, sessionID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer session.Close()

	for {
		_, _ = fmt.Fprint(os.Stdout, "👤: ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		err = rt.Run(ctx, session, line)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err = scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}