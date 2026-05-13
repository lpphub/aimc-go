package types

import (
	"fmt"

	"github.com/cloudwego/eino/schema"
)

type ApprovalResult struct {
	ApprovalID       string
	Approved         bool
	DisapproveReason *string
}

type ApprovalInfo struct {
	ToolName        string
	ArgumentsInJSON string
}

func (ai *ApprovalInfo) String() string {
	return fmt.Sprintf("⏳Approval Required\nTool: %s\nArguments: %s\nApprove? (y/n): ", ai.ToolName, ai.ArgumentsInJSON)
}

func init() {
	schema.Register[*ApprovalInfo]()
}