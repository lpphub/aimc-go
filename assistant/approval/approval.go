package approval

import (
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// Result 审批结果
type Result struct {
	ApprovalID       string  // 审批 ID，用于匹配中断
	Approved         bool
	DisapproveReason *string
}

// Info 审批信息，会展示给用户
type Info struct {
	ToolName        string
	ArgumentsInJSON string
}

func (ai *Info) String() string {
	return fmt.Sprintf("⚠️ Approval Required\nTool: %s\nArguments: %s\nApprove? (y/n): ", ai.ToolName, ai.ArgumentsInJSON)
}

func init() {
	schema.Register[*Info]()
}