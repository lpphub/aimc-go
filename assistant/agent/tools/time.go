package tools

import (
	"context"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type TimeInput struct {
	Format string `json:"format" jsonschema_description:"时间格式，如 YYYY-MM-DD 或 default"`
}

type TimeOutput struct {
	Timezone    string `json:"timezone"`
	CurrentTime string `json:"current_time"`
	Unix        int64  `json:"unix"`
}

func NewTimeTool() (tool.BaseTool, error) {
	return utils.InferTool(
		"current_time",
		"获取当前时间信息",
		func(ctx context.Context, input *TimeInput) (*TimeOutput, error) {
			now := time.Now()

			format := input.Format
			if format == "" || format == "default" {
				format = "2006-01-02 15:04:05"
			}

			return &TimeOutput{
				Timezone:    now.Location().String(),
				CurrentTime: now.Format(format),
				Unix:        now.Unix(),
			}, nil
		})
}