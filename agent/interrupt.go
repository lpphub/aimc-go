package agent

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	commontool "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino/adk"
)

func handleInterrupt(ctx context.Context, runner *adk.Runner, checkPointID string, interruptInfo *adk.InterruptInfo, scanner *bufio.Scanner) (string, error) {
	for _, ic := range interruptInfo.InterruptContexts {
		if !ic.IsRootCause {
			continue
		}

		info, ok := ic.Info.(*commontool.ApprovalInfo)
		if !ok {
			continue
		}

		fmt.Printf("\n⚠️  Approval Required ⚠️\n")
		fmt.Printf("Tool: %s\n", info.ToolName)
		fmt.Printf("Arguments: %s\n", info.ArgumentsInJSON)
		fmt.Print("\nApprove this action? (y/n): ")

		if !scanner.Scan() {
			return "", fmt.Errorf("failed to read user input")
		}
		response := strings.TrimSpace(scanner.Text())

		var resumeData *commontool.ApprovalResult
		if response == "y" || response == "yes" {
			resumeData = &commontool.ApprovalResult{Approved: true}
			fmt.Println("✓ Approved, executing...")
		} else {
			resumeData = &commontool.ApprovalResult{Approved: false}
			fmt.Println("✗ Rejected")
		}

		events, err := runner.ResumeWithParams(ctx, checkPointID, &adk.ResumeParams{
			Targets: map[string]any{
				ic.ID: resumeData,
			},
		})
		if err != nil {
			return "", fmt.Errorf("failed to resume: %w", err)
		}

		content, newInterruptInfo, err := printAndCollectAssistantWithInterruptFromEvents(events)
		if err != nil {
			return "", err
		}

		if newInterruptInfo != nil {
			return handleInterrupt(ctx, runner, checkPointID, newInterruptInfo, scanner)
		}

		return content, nil
	}

	return "", fmt.Errorf("no root cause interrupt context found")
}
