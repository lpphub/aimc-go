package llm

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

const (
	APIKey  = "sk-sp-ac76140d6ae04e939ad6b82d71c2ea31"
	Model   = "glm-5"
	BaseURL = "https://coding.dashscope.aliyuncs.com/v1"
)

func NewChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  APIKey,
		Model:   Model,
		BaseURL: BaseURL,
		ByAzure: false,
	})
}
