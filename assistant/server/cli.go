package server

import (
	"aimc-go/assistant/channel"
	"aimc-go/assistant/runtime"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

// CLIServer CLI 服务
type CLIServer struct {
	rt *runtime.Runtime
}

// NewCLIServer 创建 CLI 服务
func NewCLIServer() (*CLIServer, error) {
	rt, err := NewRuntime()
	if err != nil {
		return nil, err
	}

	return &CLIServer{rt: rt}, nil
}

// Run 运行 CLI 交互循环
func (s *CLIServer) Run() error {
	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)
	sessionID := uuid.New().String()

	ch := channel.NewCLIChannel(sessionID, scanner)
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

		err := s.rt.Run(ctx, ch, line)
		if err != nil {
			return err
		}
	}

	return scanner.Err()
}
