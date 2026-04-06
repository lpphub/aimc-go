package assistant

import (
	"aimc-go/assistant/server"
	"fmt"
	"os"
)

// RunCLI 运行 CLI 模式
func RunCLI() {
	cli, err := server.NewCLIServer()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cli.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
