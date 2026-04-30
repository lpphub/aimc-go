package runtime

import (
	"aimc-go/assistant/session"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func (r *Runtime) drain(iter *adk.AsyncIterator[*adk.AgentEvent], sess *session.Session) ([]*schema.Message, *adk.InterruptInfo, error) {
	messages := make([]*schema.Message, 0, 20)

	for {
		event, ok := iter.Next()
		if !ok {
			return messages, nil, nil
		}

		msg, interrupt, err := r.handleEvent(event, sess)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			messages = append(messages, msg)
		}
		if interrupt != nil {
			return messages, interrupt, nil
		}
	}
}

func (r *Runtime) handleEvent(event *adk.AgentEvent, sess *session.Session) (*schema.Message, *adk.InterruptInfo, error) {
	if event.Err != nil {
		//if errors.Is(event.Err, adk.ErrExceedMaxIterations) {
		//	// max iterations
		//	return nil, nil, nil
		//}
		return nil, nil, event.Err
	}

	if event.Action != nil {
		return nil, r.handleAction(event.Action, sess), nil
	}

	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil, nil, nil
	}

	return r.handleMessage(event.Output.MessageOutput, sess)
}

func (r *Runtime) handleAction(action *adk.AgentAction, sess *session.Session) *adk.InterruptInfo {
	if action.Interrupted != nil {
		return action.Interrupted
	}
	if action.TransferToAgent != nil {
		_ = sess.Emit(session.Event{
			Type:    session.TypeMessage,
			Content: fmt.Sprintf("➡️ transfer to %s\n", action.TransferToAgent.DestAgentName),
		})
		return nil
	}
	if action.Exit {
		_ = sess.Emit(session.Event{Type: session.TypeMessage, Content: "🏁 exit\n"})
	}
	return nil
}
