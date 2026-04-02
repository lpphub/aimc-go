package assistant

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/agent/llm"
	"aimc-go/assistant/agent/middleware"
	"aimc-go/assistant/agent/prompts"
	"aimc-go/assistant/approval"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

func Cli() {
	ctx := context.Background()

	// 1. model
	cm, err := llm.NewChatModel(ctx, llm.DefaultConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create model: %v\n", err)
		os.Exit(1)
	}

	// 2. tools — 使用默认工具集
	agentTools, err := agent.PresetTools(cm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to initialize tools: %v\n", err)
		os.Exit(1)
	}

	// 3. middlewares — 使用默认中间件
	middlewares, err := agent.PresetMiddlewares(ctx, cm, middleware.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to setup middlewares: %v\n", err)
		os.Exit(1)
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
		MaxIterations: 50,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create agent: %v\n", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	sink := agent.StdoutSink()

	// 5. runner — 使用默认 store/sink
	runner, err := agent.NewRunner(ag,
		agent.WithStore(agent.JSONLStore("./data/sessions")),
		agent.WithSink(sink),
		agent.WithApprovalHandler(approval.NewCLIApprovalHandler(scanner, sink)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create runner: %v\n", err)
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
