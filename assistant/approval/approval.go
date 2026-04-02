package approval

import (
	"aimc-go/assistant/sink"
	"bufio"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// ApprovalResult 审批结果
type ApprovalResult struct {
	Approved         bool
	DisapproveReason *string
}

// ApprovalHandler 抽象"如何获取用户审批结果"
// CLI: 阻塞读 stdin
// SSE: 推送事件后阻塞等待 HTTP 回调
type ApprovalHandler interface {
	GetApproval(ctx context.Context, ic *adk.InterruptCtx) (*ApprovalResult, error)
}

// ApprovalInfo 审批信息，会展示给用户
type ApprovalInfo struct {
	ToolName        string
	ArgumentsInJSON string
	InterruptID     string
}

func (ai *ApprovalInfo) String() string {
	return fmt.Sprintf("⚠️ Approval Required\nTool: %s\nArguments: %s\nApprove? (y/n): ", ai.ToolName, ai.ArgumentsInJSON)
}

func init() {
	schema.Register[*ApprovalInfo]()
}

// CLIApprovalHandler CLI 场景：阻塞读取 stdin
type CLIApprovalHandler struct {
	scanner *bufio.Scanner
	sink    sink.Sink
}

func NewCLIApprovalHandler(scanner *bufio.Scanner, s sink.Sink) *CLIApprovalHandler {
	return &CLIApprovalHandler{scanner: scanner, sink: s}
}

func (p *CLIApprovalHandler) GetApproval(_ context.Context, ic *adk.InterruptCtx) (*ApprovalResult, error) {
	info, ok := ic.Info.(*ApprovalInfo)
	if !ok {
		return nil, fmt.Errorf("unexpected interrupt info type: %T", ic.Info)
	}
	info.InterruptID = ic.ID

	p.sink.Emit(sink.Chunk{Kind: sink.KindMessage, Content: info.String()})

	if p.scanner == nil || !p.scanner.Scan() {
		return nil, fmt.Errorf("failed to read approval input")
	}
	response := strings.TrimSpace(p.scanner.Text())

	if response == "y" || response == "yes" {
		p.sink.Emit(sink.Chunk{Kind: sink.KindMessage, Content: "✔️ Approved, executing...\n"})
		return &ApprovalResult{Approved: true}, nil
	}

	p.sink.Emit(sink.Chunk{Kind: sink.KindMessage, Content: "✖️ Rejected\n"})
	return &ApprovalResult{Approved: false}, nil
}

// SSEApprovalHandler SSE 场景审批（骨架）
//
// 使用方式：
//
//	provider := NewSSEApprovalHandler(sink)
//	// Runner 内部：interrupt → provider.GetApproval() 阻塞等待
//	// HTTP handler：provider.Submit(icID, &ApprovalResult{Approved: true}) 解除阻塞
type SSEApprovalHandler struct {
	sink    sink.Sink
	pending map[string]chan *ApprovalResult
	mu      sync.Mutex
}

func NewSSEApprovalHandler(s sink.Sink) *SSEApprovalHandler {
	return &SSEApprovalHandler{
		sink:    s,
		pending: make(map[string]chan *ApprovalResult),
	}
}

func (p *SSEApprovalHandler) GetApproval(ctx context.Context, ic *adk.InterruptCtx) (*ApprovalResult, error) {
	info, ok := ic.Info.(*ApprovalInfo)
	if !ok {
		return nil, fmt.Errorf("unexpected interrupt info type: %T", ic.Info)
	}

	ch := make(chan *ApprovalResult, 1)
	p.mu.Lock()
	p.pending[ic.ID] = ch
	p.mu.Unlock()

	p.sink.Emit(sink.Chunk{Kind: sink.KindMessage, Content: info.String()})

	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		p.mu.Lock()
		delete(p.pending, ic.ID)
		p.mu.Unlock()
		return nil, ctx.Err()
	}
}

// Submit 供 HTTP handler 调用，写入审批结果解除 GetApproval 的阻塞
func (p *SSEApprovalHandler) Submit(interruptID string, result *ApprovalResult) {
	p.mu.Lock()
	ch, ok := p.pending[interruptID]
	if ok {
		delete(p.pending, interruptID)
	}
	p.mu.Unlock()

	if ok {
		ch <- result
	}
}
