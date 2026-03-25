package agent

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func printAndCollectAssistantFromEvents(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mo := event.Output.MessageOutput
		if mo.Role != schema.Assistant {
			continue
		}

		if mo.IsStreaming {
			mo.MessageStream.SetAutomaticClose()
			for {
				frame, err := mo.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return "", err
				}
				if frame != nil && frame.Content != "" {
					sb.WriteString(frame.Content)
					_, _ = fmt.Fprint(os.Stdout, frame.Content)
				}
			}
			_, _ = fmt.Fprintln(os.Stdout)
			continue
		}

		if mo.Message != nil {
			sb.WriteString(mo.Message.Content)
			_, _ = fmt.Fprintln(os.Stdout, mo.Message.Content)
		} else {
			_, _ = fmt.Fprintln(os.Stdout)
		}
	}

	return sb.String(), nil
}
