package session

import (
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// ApprovalResult 审批结果
type ApprovalResult struct {
	ApprovalID       string
	Approved         bool
	DisapproveReason *string
}

// ApprovalInfo 审批信息，会展示给用户
type ApprovalInfo struct {
	ToolName        string
	ArgumentsInJSON string
}

func (ai *ApprovalInfo) String() string {
	return fmt.Sprintf("⚠️ Approval Required\nTool: %s\nArguments: %s\nApprove? (y/n): ", ai.ToolName, ai.ArgumentsInJSON)
}

func init() {
	schema.Register[*ApprovalInfo]()
}
