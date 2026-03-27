package agent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/schema"
)

func SimpleChat() {
	query := "今天几号，郑州天气怎么样"

	ctx := context.Background()
	cm, err := newChatModel(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	messages := []*schema.Message{
		schema.SystemMessage("You are a helpful assistant."),
		schema.UserMessage(query),
	}

	_, _ = fmt.Fprint(os.Stdout, "[assistant] ")
	stream, err := cm.Stream(ctx, messages)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer stream.Close()

	for {
		frame, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if frame != nil {
			_, _ = fmt.Fprint(os.Stdout, frame.Content)
		}
	}
	_, _ = fmt.Fprintln(os.Stdout)
}

func MultiTurnChat() {
	sessionID := "5bdf84b0-f54a-46b6-aea1-b4bf4e81e8bc"

	ctx := context.Background()
	cm, _ := newChatModel(ctx)

	agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "Ch02ChatModelAgent",
		Description: "A minimal ChatModelAgent with in-memory multi-turn history.",
		Instruction: "You are a helpful assistant.",
		Model:       cm,
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	store, err := NewJSONLStore("./data/sessions")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	session, _ := store.GetOrCreate(sessionID)

	scanner := bufio.NewScanner(os.Stdin)

	for {
		_, _ = fmt.Fprint(os.Stdout, "you> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		userMsg := schema.UserMessage(line)
		_ = session.Append(userMsg)

		history := session.GetMessages()

		events := runner.Run(ctx, history)
		content, err := printAndCollectAssistantFromEvents(events)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		assistantMsg := schema.AssistantMessage(content, nil)
		_ = session.Append(assistantMsg)

		history = append(history, schema.AssistantMessage(content, nil))
	}
	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("\nSession saved: %s\n", sessionID)
}

func DeepAgent() {
	ctx := context.Background()
	cm, _ := newChatModel(ctx)

	sessionID := "5a5819ae-0225-4d09-b929-6f644e5fb897"
	projectRoot := "/home/lsk/projects/eino-demo"
	backend, _ := localbk.NewBackend(ctx, &localbk.Config{})

	// 创建 DeepAgent,自动注册文件系统工具
	prompt := fmt.Sprintf(`You are a helpful assistant that helps users learn the Eino framework.

IMPORTANT: When using filesystem tools (ls, read_file, glob, grep, etc.), you MUST use absolute paths.

The project root directory is: %s

- When the user asks to list files in "current directory", use path: %s
- When the user asks to read a file with a relative path, convert it to absolute path by prepending %s
- Example: if user says "read main.go", you should call read_file with file_path: "%s/main.go"

Always use absolute paths when calling filesystem tools.`, projectRoot, projectRoot, projectRoot, projectRoot)

	agent, err := deep.New(ctx, &deep.Config{
		Name:           "Ch04ToolAgent",
		Description:    "ChatWithDoc agent with filesystem access via LocalBackend.",
		Instruction:    prompt,
		ChatModel:      cm,
		Backend:        backend, // 提供文件系统操作能力
		StreamingShell: backend, // 提供命令执行能力
		MaxIteration:   50,
		Handlers: []adk.ChatModelAgentMiddleware{
			&safeToolMiddleware{}, // 将 Tool 错误转换为字符串
		},
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries: 3,
			IsRetryAble: func(ctx context.Context, err error) bool {
				// 429 限流错误可重试
				return strings.Contains(err.Error(), "429") ||
					strings.Contains(err.Error(), "Too Many Requests") ||
					strings.Contains(err.Error(), "qpm limit")
			},
		},
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	store, err := NewJSONLStore("./data/sessions")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	session, _ := store.GetOrCreate(sessionID)

	scanner := bufio.NewScanner(os.Stdin)

	for {
		_, _ = fmt.Fprint(os.Stdout, "you> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		userMsg := schema.UserMessage(line)
		_ = session.Append(userMsg)

		history := session.GetMessages()

		events := runner.Run(ctx, history)
		content, err := printAndCollectAssistantWithToolsFromEvents(events)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		assistantMsg := schema.AssistantMessage(content, nil)
		_ = session.Append(assistantMsg)
	}
	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
