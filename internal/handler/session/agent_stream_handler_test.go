package session

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type streamEventStub struct {
	interfaces.StreamManager
	events []interfaces.StreamEvent
}

func (s *streamEventStub) AppendEvent(_ context.Context, _, _ string, event interfaces.StreamEvent) error {
	s.events = append(s.events, event)
	return nil
}

func TestAgentStreamHandler_FailedCompletionKeepsPartialMessageIncomplete(t *testing.T) {
	stream := &streamEventStub{}
	message := &types.Message{ID: "assistant-1", Content: "partial answer"}
	h := &AgentStreamHandler{
		ctx:                context.Background(),
		sessionID:          "session-1",
		assistantMessageID: "assistant-1",
		assistantMessage:   message,
		streamManager:      stream,
	}

	err := h.handleError(context.Background(), event.Event{
		Type: event.EventError,
		Data: event.ErrorData{Error: "context deadline exceeded", Stage: "agent_execution"},
	})
	require.NoError(t, err)

	assert.Equal(t, "partial answer", message.Content)
	assert.False(t, message.IsCompleted)
	require.Len(t, stream.events, 1)
	assert.Equal(t, types.ResponseTypeError, stream.events[0].Type)
	assert.True(t, stream.events[0].Done)
	assert.Equal(t, false, stream.events[0].Data["is_completed"])

	completed := false
	err = h.handleComplete(context.Background(), event.Event{
		Type: event.EventAgentComplete,
		Data: event.AgentCompleteData{
			MessageID:   "assistant-1",
			IsCompleted: &completed,
		},
	})
	require.NoError(t, err)

	assert.False(t, message.IsCompleted)
	require.Len(t, stream.events, 2)
	assert.Equal(t, types.ResponseTypeComplete, stream.events[1].Type)
	assert.Equal(t, false, stream.events[1].Data["is_completed"])

	stoppedMessage := &types.Message{ID: "assistant-2", IsCompleted: true}
	stoppedHandler := &AgentStreamHandler{
		ctx:                context.Background(),
		sessionID:          "session-1",
		assistantMessageID: "assistant-2",
		assistantMessage:   stoppedMessage,
		streamManager:      &streamEventStub{},
	}
	require.NoError(t, stoppedHandler.handleError(context.Background(), event.Event{
		Type: event.EventError,
		Data: event.ErrorData{Error: "context canceled", Stage: "agent_execution"},
	}))
	assert.True(t, stoppedMessage.IsCompleted, "late cancellation errors must not undo user stop")
}
