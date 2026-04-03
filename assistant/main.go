package assistant

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/channel"
	"aimc-go/assistant/runtime"
	"aimc-go/assistant/store"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

// RunCLI 运行 CLI 模式
// 使用默认 Agent 配置，如需自定义可传入 agent.WithXxx() 选项
func RunCLI() {
	ctx := context.Background()

	// 创建 Agent（使用默认配置或传入的自定义选项）
	assistantAgent, err := agent.New(ctx,
		agent.WithProjectRoot("/Users/lsk/Projects/eino-demo"),
		agent.WithSkillDir("/Users/lsk/Projects/eino-demo/ext/skills"),
	)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	jsonlStore := store.NewJSONLStore("./data/sessions")

	rt, err := runtime.New(assistantAgent, runtime.WithStore(jsonlStore))
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	sessionID := uuid.New().String()

	// Channel 代表交互通道，在多轮对话期间保持
	ch := channel.NewCLIChannelBuilder(scanner).Build(sessionID)
	defer ch.Close()

	for {
		_, _ = fmt.Fprint(os.Stdout, "👤: ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		err = rt.Run(ctx, ch, line)
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
