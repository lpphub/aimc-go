package assistant

import (
	"aimc-go/assistant/agent"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

func ChatAgent() {
	ag, err := agent.New(agent.Config{
		Name:          "assistant",
		Description:   "assistant test",
		Instruction:   "You are an AI assistant that helps people find information.",
		MaxIterations: 30,
	})
	if err != nil {
		panic(err)
	}

	runner := agent.NewRunner(ag)

	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		_, _ = fmt.Fprint(os.Stdout, "you> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		_, err = runner.Run(ctx, line)
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
