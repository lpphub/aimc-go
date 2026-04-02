// assistant/command/resume.go
package command

import (
	"context"
	"fmt"
)

func ResumeCmd(ctx context.Context, deps *Dependencies, args string) error {
	if args == "" {
		// 无参数：恢复最近的会话（排除当前）
		sessions, err := deps.Store.ListSessions()
		if err != nil {
			return fmt.Errorf("获取会话列表失败: %w", err)
		}

		// 找到第一个非当前会话
		currentID := *deps.SessionID
		for _, s := range sessions {
			if s.ID != currentID {
				*deps.SessionID = s.ID
				fmt.Printf("✓ 已恢复会话: %s\n", s.ID)
				return nil
			}
		}

		return fmt.Errorf("没有其他可恢复的会话")
	}

	// 有参数：恢复指定会话
	sess, err := deps.Store.GetOrCreate(ctx, args)
	if err != nil {
		return fmt.Errorf("会话 %s 不存在: %w", args, err)
	}

	*deps.SessionID = sess.ID
	fmt.Printf("✓ 已恢复会话: %s\n", sess.ID)
	return nil
}