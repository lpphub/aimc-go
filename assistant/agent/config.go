package agent

import (
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

type Config struct {
	Name          string
	Description   string
	Instruction   string // 为空则使用默认
	MaxIterations int    // 默认 50

	Model       model.ToolCallingChatModel
	Tools       []tool.BaseTool
	Middlewares []adk.ChatModelAgentMiddleware

	ProjectRoot string // 用于 prompt 模板
	SkillDir    string // 为空则不启用 skill 中间件
	PlanTaskDir string // 为空则不启用 plan task 中间件
}

type Option func(*Config)

func WithProjectRoot(root string) Option {
	return func(c *Config) {
		c.ProjectRoot = root
	}
}

func WithSkillDir(dir string) Option {
	return func(c *Config) {
		c.SkillDir = dir
	}
}

func WithPlanTaskDir(dir string) Option {
	return func(c *Config) {
		c.PlanTaskDir = dir
	}
}

func WithModel(m model.ToolCallingChatModel) Option {
	return func(c *Config) {
		c.Model = m
	}
}

func WithTools(t []tool.BaseTool) Option {
	return func(c *Config) {
		c.Tools = t
	}
}

func WithMiddlewares(m []adk.ChatModelAgentMiddleware) Option {
	return func(c *Config) {
		c.Middlewares = m
	}
}

func defaultConfig() *Config {
	return &Config{
		Name:          "assistant",
		Description:   "AI Assistant",
		MaxIterations: 100,
	}
}
