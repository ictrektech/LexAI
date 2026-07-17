package chatpipeline

import (
	"context"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Tencent/WeKnora/internal/common"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/searchutil"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

var regThinkTags = regexp.MustCompile(`(?s)<think>.*?</think>`)

const retrievedImageOutputRequirement = `

## Retrieved Image Output Requirement
The retrieved context for this turn contains Markdown images. Images attached to retrieved passages should be treated as relevant by default.
- Unless the user explicitly requests text-only output, or every retrieved image is clearly unrelated to the answer, the final answer MUST include at least one relevant Markdown image copied from the retrieved context.
- Copy the complete Markdown image syntax and its URL verbatim. Never invent, shorten, normalize, or replace the URL.
- Use ASCII half-width parentheses in image Markdown exactly as ![alt](url). Never use full-width （ or ）.
- Place each image immediately after the paragraph it supports, rather than collecting images at the end.
- When multiple retrieved images support different sections of a multi-section answer, include them in their corresponding sections instead of stopping after the first image.
- Before finishing, silently verify that the answer contains a Markdown image whenever this requirement applies.`

func appendRetrievedImageOutputRequirement(systemPrompt, renderedContexts string) string {
	if !searchutil.MarkdownImageRegex.MatchString(renderedContexts) {
		return systemPrompt
	}
	return strings.TrimRight(systemPrompt, " \t\r\n") + retrievedImageOutputRequirement
}

// pipelineInfo logs pipeline info level entries.
func pipelineInfo(ctx context.Context, stage, action string, fields map[string]interface{}) {
	common.PipelineInfo(ctx, stage, action, fields)
}

// pipelineWarn logs pipeline warning level entries.
func pipelineWarn(ctx context.Context, stage, action string, fields map[string]interface{}) {
	common.PipelineWarn(ctx, stage, action, fields)
}

// pipelineError logs pipeline error level entries.
func pipelineError(ctx context.Context, stage, action string, fields map[string]interface{}) {
	common.PipelineError(ctx, stage, action, fields)
}

// prepareChatModel shared logic to prepare chat model and options
// it gets the chat model and sets up the chat options based on the chat manage.
func prepareChatModel(ctx context.Context, modelService interfaces.ModelService,
	chatManage *types.ChatManage,
) (chat.Chat, *chat.ChatOptions, error) {
	chatModel, err := modelService.GetChatModel(ctx, chatManage.ChatModelID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get chat model: %v", err)
		return nil, nil, err
	}

	opt := &chat.ChatOptions{
		Temperature:         chatManage.SummaryConfig.Temperature,
		TopP:                chatManage.SummaryConfig.TopP,
		Seed:                chatManage.SummaryConfig.Seed,
		MaxTokens:           chatManage.SummaryConfig.MaxTokens,
		MaxCompletionTokens: chatManage.SummaryConfig.MaxCompletionTokens,
		FrequencyPenalty:    chatManage.SummaryConfig.FrequencyPenalty,
		PresencePenalty:     chatManage.SummaryConfig.PresencePenalty,
		Thinking:            chatManage.SummaryConfig.Thinking,
	}
	if opt.Thinking != nil {
		pipelineInfo(ctx, "Stream", "thinking_option", map[string]interface{}{
			"enabled": *opt.Thinking,
		})
	}

	return chatModel, opt, nil
}

// prepareMessagesWithHistory prepare complete messages including history.
// When SystemPromptOverride is set (e.g. by intent-specific prompt logic),
// it takes precedence over the default SummaryConfig.Prompt.
func prepareMessagesWithHistory(chatManage *types.ChatManage, opt *chat.ChatOptions) []chat.Message {
	base := chatManage.SummaryConfig.Prompt
	if chatManage.SystemPromptOverride != "" {
		base = chatManage.SystemPromptOverride
	}
	renderMessages := func(contexts string) []chat.Message {
		systemPrompt := types.RenderPromptPlaceholders(base, types.PlaceholderValues{
			"query":    chatManage.Query,
			"language": chatManage.Language,
			"contexts": contexts,
		})
		systemPrompt = appendRetrievedImageOutputRequirement(systemPrompt, contexts)

		chatMessages := []chat.Message{
			{Role: "system", Content: systemPrompt},
		}

		chatMessages = AppendHistoryMessages(chatMessages, chatManage.History)

		// Add current user message. Only include images when the chat model supports
		// vision; non-vision models rely on the text description in UserContent.
		userMsg := chat.Message{Role: "user", Content: chatManage.UserContent}
		if chatManage.ChatModelSupportsVision && len(chatManage.Images) > 0 {
			userMsg.Images = chatManage.Images
		}
		return append(chatMessages, userMsg)
	}

	chatMessages := renderMessages(chatManage.RenderedContexts)
	inputBudget := chatPromptInputBudget(opt)
	if inputBudget <= 0 || estimateMessagesTokens(chatMessages) <= inputBudget ||
		strings.TrimSpace(chatManage.RenderedContexts) == "" {
		return chatMessages
	}

	truncatedContexts := truncateContextsToBudget(chatManage.RenderedContexts, inputBudget, renderMessages)
	if truncatedContexts != chatManage.RenderedContexts {
		chatManage.RenderedContexts = truncatedContexts
		chatMessages = renderMessages(truncatedContexts)
	}
	return chatMessages
}

func chatPromptInputBudget(opt *chat.ChatOptions) int {
	maxContext := envInt("WEKNORA_CHAT_MODEL_CONTEXT_TOKENS", 16384)
	if maxContext <= 0 {
		return 0
	}
	outputTokens := 2048
	if opt != nil {
		if opt.MaxTokens > outputTokens {
			outputTokens = opt.MaxTokens
		}
		if opt.MaxCompletionTokens > outputTokens {
			outputTokens = opt.MaxCompletionTokens
		}
	}
	safety := envInt("WEKNORA_CHAT_CONTEXT_SAFETY_TOKENS", 768)
	return maxContext - outputTokens - safety
}

func truncateContextsToBudget(
	contexts string,
	inputBudget int,
	renderMessages func(string) []chat.Message,
) string {
	runes := []rune(contexts)
	if len(runes) == 0 {
		return contexts
	}

	lo, hi := 0, len(runes)
	best := ""
	for lo <= hi {
		mid := (lo + hi) / 2
		candidate := strings.TrimSpace(string(runes[:mid]))
		if candidate != "" && mid < len(runes) {
			candidate += "\n\n<note>Retrieved context was truncated to fit the model context window.</note>"
		}
		if estimateMessagesTokens(renderMessages(candidate)) <= inputBudget {
			best = candidate
			lo = mid + 1
			continue
		}
		hi = mid - 1
	}
	return best
}

func estimateMessagesTokens(messages []chat.Message) int {
	total := 0
	for _, msg := range messages {
		total += estimatePromptTokens(msg.Role)
		total += estimatePromptTokens(msg.Content)
		for _, image := range msg.Images {
			total += estimatePromptTokens(image)
		}
		// Add a small per-message overhead for chat formatting.
		total += 8
	}
	return total
}

func estimatePromptTokens(text string) int {
	if text == "" {
		return 0
	}
	// Conservative estimator: every visible rune counts as one token. This
	// intentionally overestimates English/XML and is close enough for Chinese
	// legal text, so the final prompt stays below the remote model hard limit.
	count := 0
	for _, r := range text {
		if r == '\n' || r == '\t' || r == ' ' || r == '\r' {
			continue
		}
		count++
	}
	return count
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// AppendHistoryMessages appends prior Q&A rounds in chronological order.
// History is already filtered and truncated upstream by the load_history plugin.
func AppendHistoryMessages(messages []chat.Message, history []*types.History) []chat.Message {
	for _, history := range history {
		messages = append(messages, chat.Message{Role: "user", Content: history.Query})
		messages = append(messages, chat.Message{Role: "assistant", Content: history.Answer})
	}
	return messages
}

// loadAndProcessHistory fetches recent messages, groups them into Q&A pairs,
// strips <think> tags from assistant answers, sorts by recency, and limits to maxRounds.
// fetchCount controls how many raw messages to fetch (typically maxRounds*2+10).
func loadAndProcessHistory(
	ctx context.Context,
	messageService interfaces.MessageService,
	sessionID string,
	maxRounds int,
	fetchCount int,
) ([]*types.History, error) {
	history, err := messageService.GetRecentMessagesBySession(ctx, sessionID, fetchCount)
	if err != nil {
		return nil, err
	}

	historyMap := make(map[string]*types.History)
	for _, message := range history {
		h, ok := historyMap[message.RequestID]
		if !ok {
			h = &types.History{}
		}
		if message.Role == "user" {
			// RenderedContent is a snapshot of the prompt/context format used by
			// the original turn. Replaying it would mix legacy <context id="…">
			// envelopes and old citation instructions into the current protocol.
			// Historical references are carried separately in KnowledgeReferences
			// and can be re-merged into this turn's freshly rendered context.
			h.Query = message.Content
			h.CreateAt = message.CreatedAt
			if desc := extractImageCaptions(message.Images); desc != "" {
				h.Query += "\n\n[用户上传图片内容]\n" + desc
			}
			if len(message.Attachments) > 0 {
				h.Query += message.Attachments.BuildPrompt()
			}
		} else {
			h.Answer = regThinkTags.ReplaceAllString(message.Content, "")
			h.KnowledgeReferences = message.KnowledgeReferences
		}
		historyMap[message.RequestID] = h
	}

	historyList := make([]*types.History, 0, len(historyMap))
	for _, h := range historyMap {
		if h.Answer != "" && h.Query != "" {
			historyList = append(historyList, h)
		}
	}

	sort.Slice(historyList, func(i, j int) bool {
		return historyList[i].CreateAt.After(historyList[j].CreateAt)
	})

	if len(historyList) > maxRounds {
		historyList = historyList[:maxRounds]
	}

	slices.Reverse(historyList)
	return historyList, nil
}

// extractImageCaptions concatenates non-empty Caption fields from stored
// message images. Used when loading history so that previous turns' image
// descriptions are visible to the model.
func extractImageCaptions(images types.MessageImages) string {
	var parts []string
	for _, img := range images {
		if img.Caption != "" {
			parts = append(parts, img.Caption)
		}
	}
	return strings.Join(parts, "\n")
}

// ---------------------------------------------------------------------------
// Concurrency utilities
// ---------------------------------------------------------------------------

// ParallelTask represents a named unit of concurrent work.
type ParallelTask struct {
	Name string
	Run  func() *PluginError
}

// RunParallel executes tasks concurrently.
// Returns a map of task name → error for tasks that returned non-nil errors.
func RunParallel(tasks ...ParallelTask) map[string]*PluginError {
	errs := make(map[string]*PluginError)
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(len(tasks))
	for _, task := range tasks {
		go func(t ParallelTask) {
			defer wg.Done()
			if err := t.Run(); err != nil {
				mu.Lock()
				errs[t.Name] = err
				mu.Unlock()
			}
		}(task)
	}
	wg.Wait()
	return errs
}

// ParallelMap applies fn to each element of items concurrently (up to
// maxWorkers goroutines) and returns results in the same order as items.
// If maxWorkers <= 0, concurrency is unbounded (one goroutine per item).
func ParallelMap[T, R any](items []T, maxWorkers int, fn func(int, T) R) []R {
	n := len(items)
	if n == 0 {
		return nil
	}
	results := make([]R, n)

	if maxWorkers <= 0 || maxWorkers > n {
		maxWorkers = n
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)

	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, it T) {
			defer func() { <-sem; wg.Done() }()
			results[idx] = fn(idx, it)
		}(i, item)
	}
	wg.Wait()
	return results
}
