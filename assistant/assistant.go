package assistant

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/agent/llm"
	"aimc-go/assistant/agent/middleware"
	"aimc-go/assistant/agent/prompts"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

func Agent() {
	ctx := context.Background()

	// 1. model
	cm, err := llm.NewChatModel(ctx, llm.DefaultConfig())
	if err != nil {
		panic(err)
	}

	// 2. tools — 使用默认工具集
	agentTools, err := agent.DefaultTools(cm)
	if err != nil {
		panic(err)
	}

	// 3. middlewares — 使用默认中间件
	middlewares, err := agent.DefaultMiddlewares(ctx, cm, middleware.Config{})
	if err != nil {
		panic(err)
	}

	// 4. agent
	projectRoot := "/home/lsk/projects/eino-demo"
	ag, err := agent.New(ctx, agent.AgentConfig{
		Name:          "enio-assistant",
		Description:   "enio tutorial assistant",
		Instruction:   fmt.Sprintf(prompts.EinoTutorial, projectRoot, projectRoot, projectRoot, projectRoot),
		Model:         cm,
		Tools:         agentTools,
		Middlewares:   middlewares,
		MaxIterations: 30,
	})
	if err != nil {
		panic(err)
	}

	// 5. runner — 使用默认 store/sink
	runner, err := agent.NewRunner(ag,
		agent.WithStore(agent.DefaultStore("./data/sessions")),
		agent.WithSink(agent.DefaultSink()),
	)
	if err != nil {
		panic(err)
	}

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
