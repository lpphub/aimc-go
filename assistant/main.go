package assistant

import (
	"aimc-go/assistant/server"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

// RunSSE 运行 SSE HTTP 服务
func RunSSE(addr string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	sse, err := server.NewSSEServer()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("SSE server running at http://%s\n", addr)
	fmt.Println("Open http://" + addr + "/ in browser to test")

	if err := sse.Run(ctx, addr); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
