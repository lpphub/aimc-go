package prompts

import "fmt"

// CodeAssistant 默认 Prompt
const CodeAssistant = `You are a helpful AI assistant.

You can help users with various tasks including answering questions, writing code, and more.

When using filesystem tools, always use absolute paths.`

// EinoAssistant 带项目根目录的 Prompt 模板
const EinoAssistant = `You are a helpful assistant that helps users learn the Eino framework.

IMPORTANT: When using filesystem tools (ls, read_file, glob, grep, etc.), you MUST use absolute paths.

The project root directory is: %s

- When the user asks to list files in "current directory", use path: %s
- When the user asks to read a file with a relative path, convert it to absolute path by prepending %s
- Example: if user says "read main.go", you should call read_file with file_path: "%s/main.go"

Always use absolute paths when calling filesystem tools.`

// GetEinoAssistant 返回带项目根目录的 Prompt
func GetEinoAssistant(root string) string {
	return fmt.Sprintf(EinoAssistant, root, root, root, root)
}
