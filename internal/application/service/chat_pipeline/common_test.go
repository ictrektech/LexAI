package chatpipeline

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

func TestPrepareMessagesWithHistoryTruncatesRenderedContexts(t *testing.T) {
	t.Setenv("WEKNORA_CHAT_MODEL_CONTEXT_TOKENS", "18432")
	t.Setenv("WEKNORA_CHAT_CONTEXT_SAFETY_TOKENS", "768")

	cm := &types.ChatManage{
		PipelineRequest: types.PipelineRequest{
			Query:    "这个问题需要检索",
			Language: "zh-CN",
			SummaryConfig: types.SummaryConfig{
				Prompt: "{{contexts}}\n请回答：{{query}}",
			},
		},
		PipelineState: types.PipelineState{
			RenderedContexts: strings.Repeat("中华人民共和国民法典合同编相关条文。", 2000),
			UserContent:      "这个问题需要检索",
		},
	}

	messages := prepareMessagesWithHistory(cm, &chat.ChatOptions{MaxCompletionTokens: 2048})
	budget := chatPromptInputBudget(&chat.ChatOptions{MaxCompletionTokens: 2048})
	if got := estimateMessagesTokens(messages); got > budget {
		t.Fatalf("estimated prompt tokens = %d, want <= %d", got, budget)
	}
	if !strings.Contains(cm.RenderedContexts, "Retrieved context was truncated") {
		t.Fatal("expected rendered contexts to include truncation note")
	}
}
