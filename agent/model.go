package agent

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

const (
	APIKey  = "sk-2ec2c691723e4b07a6577bc9e818af09"
	Model   = "deepseek-chat"
	BaseURL = "https://api.deepseek.com"

	APIKeyBL  = "sk-sp-ac76140d6ae04e939ad6b82d71c2ea31"
	ModelBL   = "qwen3-max-2026-01-23"
	BaseURLBL = "https://coding.dashscope.aliyuncs.com/v1"
)

func newChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  APIKeyBL,
		Model:   ModelBL,
		BaseURL: BaseURLBL,
		ByAzure: false,
	})
}
