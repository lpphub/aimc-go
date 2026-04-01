package assistant

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/agent/llm"
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

	// 1. 创建 model（调用方决定配置）
	cm, err := llm.NewChatModel(ctx, llm.DefaultConfig())
	if err != nil {
		panic(err)
	}

	// 2. 创建 agent（依赖注入，tools/middlewares 未指定则用默认值）
	projectRoot := "/home/lsk/projects/eino-demo"
	ag, err := agent.New(ctx, agent.AgentConfig{
		Name:          "enio-assistant",
		Description:   "enio tutorial assistant",
		Instruction:   fmt.Sprintf(prompts.EinoTutorial, projectRoot, projectRoot, projectRoot, projectRoot),
		Model:         cm,
		MaxIterations: 30,
	})
	if err != nil {
		panic(err)
	}

	// 3. 创建 runner（未指定 store/sink 则用默认值）
	runner := agent.NewRunner(ag)

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
