package runtime

import "github.com/cloudwego/eino/schema"

func trimRounds(history []*schema.Message, maxRounds int) []*schema.Message {
	if maxRounds <= 0 || len(history) == 0 {
		return history
	}

	keepStart := 0
	userCount := 0
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == schema.User {
			userCount++
			if userCount == maxRounds {
				keepStart = i
				break
			}
		}
	}

	if keepStart == 0 {
		return history
	}

	result := make([]*schema.Message, len(history)-keepStart)
	copy(result, history[keepStart:])
	return result
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}