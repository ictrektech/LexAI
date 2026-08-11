package session

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
)

type completionMessageStub struct {
	interfaces.MessageService
	updated    *types.Message
	updateCall int
	indexCall  int
}

func (s *completionMessageStub) UpdateMessage(_ context.Context, message *types.Message) error {
	s.updateCall++
	s.updated = message
	return nil
}

func (s *completionMessageStub) IndexMessageToKB(context.Context, string, string, string, string) {
	s.indexCall++
}

func TestTagScopesFromMentionedItems(t *testing.T) {
	scopes := tagScopesFromMentionedItems([]MentionedItemRequest{
		{Type: "tag", ID: "tag-1", KBID: "kb-1"},
		{Type: "tag", ID: "tag-2", KBID: "kb-1"},
		{Type: "tag", ID: "tag-3", KBID: "kb-2"},
		{Type: "tag", ID: "orphan", KBID: ""},
	})
	assert.Len(t, scopes, 2)
	byKB := make(map[string][]string)
	for _, scope := range scopes {
		byKB[scope.KnowledgeBaseID] = scope.TagIDs
	}
	assert.ElementsMatch(t, []string{"tag-1", "tag-2"}, byKB["kb-1"])
	assert.Equal(t, []string{"tag-3"}, byKB["kb-2"])
}

func TestMergeTagScopesFromRequestIDs_SingleKB(t *testing.T) {
	scopes := mergeTagScopesFromRequestIDs(
		[]types.TagScope{{KnowledgeBaseID: "kb-1", TagIDs: []string{"tag-1"}}},
		[]string{"tag-2"},
		[]string{"kb-1"},
	)
	assert.Len(t, scopes, 1)
	assert.ElementsMatch(t, []string{"tag-1", "tag-2"}, scopes[0].TagIDs)
}

func TestMergeTagScopesFromRequestIDs_OrphanWithSingleKB(t *testing.T) {
	scopes := mergeTagScopesFromRequestIDs(nil, []string{"tag-9"}, []string{"kb-1"})
	assert.Len(t, scopes, 1)
	assert.Equal(t, "kb-1", scopes[0].KnowledgeBaseID)
	assert.Equal(t, []string{"tag-9"}, scopes[0].TagIDs)
}

func TestMergeTagScopesFromRequestIDs_AmbiguousKBIgnored(t *testing.T) {
	scopes := mergeTagScopesFromRequestIDs(nil, []string{"tag-9"}, []string{"kb-1", "kb-2"})
	assert.Empty(t, scopes)
}

func TestValidateUnscopedTagIDs(t *testing.T) {
	assert.NoError(t, validateUnscopedTagIDs(nil, nil))
	assert.NoError(t, validateUnscopedTagIDs(nil, []string{"kb-1", "kb-2"}))
	assert.NoError(t, validateUnscopedTagIDs([]string{"tag-9"}, []string{"kb-1"}))
	assert.Error(t, validateUnscopedTagIDs([]string{"tag-9"}, []string{"kb-1", "kb-2"}))
	assert.Error(t, validateUnscopedTagIDs([]string{"tag-9"}, nil))
}

func TestCompleteAssistantMessage_PartialFailureKeepsContentAndSkipsDerivedWork(t *testing.T) {
	messageService := &completionMessageStub{}
	h := &Handler{messageService: messageService}
	message := &types.Message{
		ID:        "assistant-1",
		SessionID: "session-1",
		Content:   "partial answer",
		Role:      "assistant",
	}

	h.completeAssistantMessage(context.Background(), message, "审查合同", false)

	assert.Equal(t, 1, messageService.updateCall)
	assert.Same(t, message, messageService.updated)
	assert.Equal(t, "partial answer", message.Content)
	assert.False(t, message.IsCompleted)
	assert.Equal(t, 0, messageService.indexCall)
}
