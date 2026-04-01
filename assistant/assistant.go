package assistant

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/agent/prompts"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

func Agent() {
	projectRoot := "/home/lsk/projects/eino-demo"

	ag, err := agent.New(agent.Config{
		Name:          "enio-assistant",
		Description:   "enio tutorial assistant",
		Instruction:   fmt.Sprintf(prompts.EinoTutorial, projectRoot, projectRoot, projectRoot, projectRoot),
		MaxIterations: 30,
	})
	if err != nil {
		panic(err)
	}

	runner := agent.NewRunner(ag)

	ctx := context.Background()
	sessionID := uuid.New().String()

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

		_, err = runner.Run(ctx, sessionID, line)
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
