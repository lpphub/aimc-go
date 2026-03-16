package model

import (
	"context"
)

type GenerationType int8

const (
	Text GenerationType = iota
	Image
)

type AIInput struct {
	GenType GenerationType
	Prompt  string
}
type AIOutput struct {
	Text string // 文本生成
	URL  string // 图片/视频 URL
}

// AIClient 是统一接口
type AIClient interface {
	Generate(ctx context.Context, input AIInput) (*AIOutput, error)
}
