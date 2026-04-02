// assistant/command/quit.go
package command

import (
	"context"
	"fmt"
	"os"
)

func QuitCmd(ctx context.Context, deps *Dependencies, args string) error {
	fmt.Println("再见！")
	os.Exit(0)
	return nil
}