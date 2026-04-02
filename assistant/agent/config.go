package agent

import (
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

// Config Agent 配置
type Config struct {
	// Agent 基础配置
	Name          string
	Description   string
	MaxIterations int // 0 默认 50

	// 路径配置
	ProjectRoot string // 项目根目录，用于 prompt 模板
	SkillDir    string // skill 目录，为空则不启用 skill 中间件

	// 可选覆盖（不设置则使用默认）
	Model       model.ToolCallingChatModel
	Tools       []tool.BaseTool
	Middlewares []adk.ChatModelAgentMiddleware
}

// Option 配置选项
type Option func(*Config)

// WithProjectRoot 设置项目根目录
func WithProjectRoot(root string) Option {
	return func(c *Config) {
		c.ProjectRoot = root
	}
}

// WithSkillDir 设置 skill 目录
func WithSkillDir(dir string) Option {
	return func(c *Config) {
		c.SkillDir = dir
	}
}

// WithModel 设置自定义模型
func WithModel(m model.ToolCallingChatModel) Option {
	return func(c *Config) {
		c.Model = m
	}
}

// WithTools 设置自定义工具集
func WithTools(t []tool.BaseTool) Option {
	return func(c *Config) {
		c.Tools = t
	}
}

// WithMiddlewares 设置自定义中间件
func WithMiddlewares(m []adk.ChatModelAgentMiddleware) Option {
	return func(c *Config) {
		c.Middlewares = m
	}
}

func defaultConfig() *Config {
	return &Config{
		Name:          "assistant",
		Description:   "AI Assistant",
		MaxIterations: 50,
	}
}