// assistant/command/command.go
package command

import (
	"aimc-go/assistant/agent"
	"aimc-go/assistant/store"
	"bufio"
	"context"
	"fmt"
	"strings"
)

// Command 命令函数签名
type Command func(ctx context.Context, deps *Dependencies, args string) error

// Dependencies 命令执行依赖
type Dependencies struct {
	Store     store.Store
	Runner    *agent.Runner
	SessionID *string // 指向当前 session ID
	Scanner   *bufio.Scanner
}

// Registry 命令注册表
var Registry = map[string]Command{
	"new":    NewCmd,
	"resume": ResumeCmd,
	"quit":   QuitCmd,
}

// Parse 解析输入，返回命令名和参数
func Parse(input string) (name string, args string) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return "", ""
	}

	input = strings.TrimPrefix(input, "/")
	parts := strings.SplitN(input, " ", 2)
	name = parts[0]
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}
	return name, args
}

// Execute 执行命令
func Execute(ctx context.Context, deps *Dependencies, name, args string) error {
	cmd, ok := Registry[name]
	if !ok {
		return fmt.Errorf("未知命令: /%s", name)
	}
	return cmd(ctx, deps, args)
}