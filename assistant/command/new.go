// assistant/command/new.go
package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func NewCmd(ctx context.Context, deps *Dependencies, args string) error {
	newID := uuid.New().String()
	*deps.SessionID = newID
	fmt.Printf("✓ 已切换到新会话: %s\n", newID)
	return nil
}