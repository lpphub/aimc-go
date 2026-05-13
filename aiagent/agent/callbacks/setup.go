package callbacks

import (
	einocallbacks "github.com/cloudwego/eino/callbacks"
)

type Setup struct {
	UsageStats *UsageStats
}

func Init() *Setup {
	stats := &UsageStats{}
	einocallbacks.AppendGlobalHandlers(
		NewUsageHandler(),
	)
	return &Setup{UsageStats: stats}
}
