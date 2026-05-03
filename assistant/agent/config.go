package agent

import (
	"github.com/cloudwego/eino/components/model"
)

type Config struct {
	Name          string
	Description   string
	Instruction   string // 为空则使用默认 prompt
	MaxIterations int    // 默认 100

	ProjectRoot string // 用于 prompt 模板
	SkillDir    string // 为空则不启用 skill 中间件
	PlanTaskDir string // 为空则不启用 plan task 中间件

	Model model.ToolCallingChatModel // 可选，为空则自动创建
}

type Option func(*Config)

func WithName(name string) Option {
	return func(c *Config) { c.Name = name }
}

func WithDescription(desc string) Option {
	return func(c *Config) { c.Description = desc }
}

func WithInstruction(s string) Option {
	return func(c *Config) { c.Instruction = s }
}

func WithMaxIterations(n int) Option {
	return func(c *Config) { c.MaxIterations = n }
}

func WithProjectRoot(root string) Option {
	return func(c *Config) { c.ProjectRoot = root }
}

func WithSkillDir(dir string) Option {
	return func(c *Config) { c.SkillDir = dir }
}

func WithPlanTaskDir(dir string) Option {
	return func(c *Config) { c.PlanTaskDir = dir }
}

func WithModel(m model.ToolCallingChatModel) Option {
	return func(c *Config) { c.Model = m }
}

func defaultConfig() *Config {
	return &Config{
		Name:          "assistant",
		Description:   "AI Assistant",
		MaxIterations: 100,
	}
}