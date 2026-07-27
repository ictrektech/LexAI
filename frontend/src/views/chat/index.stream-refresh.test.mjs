import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { test } from 'node:test'
import assert from 'node:assert/strict'

const source = readFileSync(join(import.meta.dirname, 'index.vue'), 'utf8')

test('stream message updates refresh the current row', () => {
  assert.match(source, /const refreshMessageRow = \(message\) => \{/)
  assert.match(source, /messagesList\.splice\(index,\s*1,\s*message\)/)
  assert.match(source, /onMessageUpdated:\s*\(message,\s*payload\) => \{[\s\S]*refreshMessageRow\(message\)/)
})

test('completed quick answers sync references without page navigation', () => {
  assert.match(source, /const syncCompletedMessageReferences = \(message,\s*attempt = 0\) => \{/)
  assert.match(source, /getMessageList\(\{ session_id: targetSessionId[\s\S]*fresh\.knowledge_references\.slice\(\)/)
  assert.match(source, /if \(payload\?\.is_completed\) \{[\s\S]*syncCompletedMessageReferences\(message\)/)
})

test('completed quick answer reference sync retries and tolerates stream id drift', () => {
  assert.match(source, /const findFreshMessageForReferences = \(items,\s*message\) => \{/)
  assert.match(source, /String\(item\.content \|\| ''\)\.trim\(\) === targetContent/)
  assert.match(source, /if \(!fresh\?\.knowledge_references\?\.length\) \{[\s\S]*attempt < 10[\s\S]*syncCompletedMessageReferences\(message,\s*attempt \+ 1\)/)
})

const handlerSource = readFileSync(
  join(import.meta.dirname, '../../composables/useChatStreamHandler.ts'),
  'utf8',
)
const agentStreamSource = readFileSync(
  join(import.meta.dirname, 'components/AgentStreamDisplay.vue'),
  'utf8',
)

test('final stream updates replace placeholder references', () => {
  assert.match(handlerSource, /const refs = extractKnowledgeReferences\(payload\)/)
  assert.match(handlerSource, /if \(refs\.length > 0\) \{[\s\S]*message\.knowledge_references = refs\.slice\(\)/)
  assert.doesNotMatch(handlerSource, /if \(!message\.knowledge_references\) \{[\s\S]*message\.knowledge_references = payload\.knowledge_references/)
})

test('stream handler accepts pull-stream type field for references', () => {
  assert.match(handlerSource, /const getChunkType = \(data: ChatMessage\) =>[\s\S]*data\.response_type \|\| data\.type/)
  assert.match(handlerSource, /const responseType = getChunkType\(data\)[\s\S]*if \(responseType === 'references'\)/)
})

test('pre-answer references are kept for the first answer row', () => {
  assert.match(handlerSource, /let pendingKnowledgeReferences: unknown\[\] = \[\]/)
  assert.match(handlerSource, /pendingKnowledgeReferences = refs\.slice\(\)[\s\S]*return undefined/)
  assert.match(handlerSource, /entry\.knowledge_references = pendingKnowledgeReferences\.slice\(\)/)
})

test('completed stream rows merge with refreshed history when ids drift', () => {
  assert.match(handlerSource, /const findCurrentTurnAssistantByContent = \(item: ChatMessage\) => \{/)
  assert.match(handlerSource, /if \(message\.role === 'user'\) break/)
  assert.match(handlerSource, /const existing = findExistingMessage\(item,\s*!isScrollType\)/)
  assert.match(handlerSource, /const mergeHistoryMessage = \(existing: ChatMessage, item: ChatMessage\) => \{/)
  assert.match(handlerSource, /message = findCurrentTurnAssistantByContent\(\{\s*\.\.\.payload,\s*role: 'assistant',\s*\}\)/)
})

test('history refresh preserves active stream ids and later chunks target that row', () => {
  assert.match(handlerSource, /const streamId = existing\.id/)
  assert.match(handlerSource, /const streamRequestId = existing\.request_id/)
  assert.match(handlerSource, /if \(streamId\) existing\.id = streamId/)
  assert.match(handlerSource, /if \(streamRequestId\) existing\.request_id = streamRequestId/)
  assert.match(handlerSource, /const activeAssistantMessageId = currentAssistantMessageId\.value/)
  assert.match(handlerSource, /item\.id === activeAssistantMessageId[\s\S]*item\.request_id === activeAssistantMessageId/)
})

test('completed assistant rows are deduped inside the current user turn', () => {
  assert.match(handlerSource, /const dedupeCurrentTurnCompletedAssistants = \(preferred\?: ChatMessage\) => \{/)
  assert.match(handlerSource, /const lastUserIndex = findLastUserMessageIndex\(\)/)
  assert.match(handlerSource, /const candidates: ChatMessage\[\] = \[\]/)
  assert.match(handlerSource, /if \(message\.role !== 'assistant'\) continue/)
  assert.match(handlerSource, /if \(message\.is_completed\) score \+= 1000/)
  assert.match(handlerSource, /mergeAssistantRuntimeState\(retained,\s*message\)/)
  assert.match(handlerSource, /messagesList\.splice\(i,\s*1\)/)
  assert.match(handlerSource, /message = dedupeCurrentTurnCompletedAssistants\(message\) \|\| message/)
  assert.match(handlerSource, /const retainedEntry = payload\.is_completed[\s\S]*dedupeCurrentTurnCompletedAssistants\(entry\) \|\| entry/)
  assert.match(handlerSource, /messagesList\.push\(\.\.\.processed\)[\s\S]*dedupeCurrentTurnCompletedAssistants\(\)/)
})

test('answer stream chunks merge snapshots without duplicating the same answer', () => {
  assert.match(handlerSource, /const mergeStreamText = \(currentValue: unknown,\s*incomingValue: unknown\) => \{/)
  assert.match(handlerSource, /if \(current === incoming\) return current/)
  assert.match(handlerSource, /if \(incoming\.startsWith\(current\)\) return incoming/)
  assert.match(handlerSource, /if \(current\.startsWith\(incoming\)\) return current/)
  assert.match(handlerSource, /if \(current\.includes\(incoming\)\) return current/)
  assert.match(handlerSource, /if \(incoming\.includes\(current\)\) return incoming/)
  assert.match(handlerSource, /if \(current\.endsWith\(incoming\)\) return current/)
  assert.match(handlerSource, /message\.content = mergeStreamText\(message\.content,\s*payload\.content\)/)
  assert.doesNotMatch(handlerSource, /message\.content = payload\.content/)
  assert.match(handlerSource, /answerEvent\.content = mergeStreamText\(answerEvent\.content,\s*data\.content\)/)
  assert.match(handlerSource, /if \(data\.content\) \{[\s\S]*\} else if \(!answerEvent\.content && message\.content/)
})

test('chat view renders a deduped message list', () => {
  assert.match(source, /v-for=\"\(session, index\) in renderedMessagesList\"/)
  assert.match(source, /:user-query="getRenderedUserQuery\(index\)"/)
  assert.match(source, /const getRenderedUserQuery = \(index\) => \{[\s\S]*renderedMessagesList\.value\[i\]/)
  assert.match(source, /const renderedMessagesList = computed\(\(\) => \{/)
  assert.match(source, /let currentTurnAssistantIndex = -1/)
  assert.match(source, /const getRenderedMessageAnswerText = \(message\) => \{/)
  assert.match(source, /message\.role === 'assistant'[\s\S]*normalizeRenderedMessageContent\(getRenderedMessageAnswerText\(message\)\)/)
  assert.match(source, /if \(message\?\.is_completed\) score \+= 1000/)
  assert.match(source, /result\[currentTurnAssistantIndex\] = message/)
})

test('completed background streams clear their own session cache', () => {
  assert.match(handlerSource, /const getChunkSessionId = \(data: ChatMessage\) =>/)
  assert.match(handlerSource, /message\.__stream_session_id = streamSessionId/)
  assert.match(source, /const getMessageStreamSessionId = \(message\) =>/)
  assert.match(source, /const completedSessionId = getMessageStreamSessionId\(message\) \|\| session_id\.value/)
  assert.match(source, /inFlightTurnCache\.delete\(completedSessionId\)/)
  assert.match(source, /if \(completedSessionId === session_id\.value\) \{[\s\S]*loadFollowUpSuggestions\(message,\s*true\)/)
})

test('restored in-flight cache is discarded when history already has the completed answer', () => {
  assert.match(source, /const hasPersistedAssistantAfterCachedUser = \(cachedUserMessage\) => \{/)
  assert.match(source, /message\?\.role === 'assistant' &&[\s\S]*normalizeAssistantAnswerText\(getAssistantAnswerText\(message\)\)/)
  assert.match(source, /if \(!hasActiveStream\(targetSessionId\) && hasPersistedAssistantAfterCachedUser\(cachedUserMessage\)\) \{[\s\S]*inFlightTurnCache\.delete\(targetSessionId\)/)
})

test('rag answer display dedupes identical answer events', () => {
  assert.match(agentStreamSource, /return dedupeAnswerEvents\(result\.filter\(\(e: any\) => e\.type === 'answer'\)\)/)
  assert.match(agentStreamSource, /const dedupeAnswerEvents = \(events: any\[\]\): any\[\] => \{/)
  assert.match(agentStreamSource, /const indexByContent = new Map<string, number>\(\)/)
  assert.match(agentStreamSource, /retained\[existingIndex\] = \{[\s\S]*done: Boolean\(retained\[existingIndex\]\?\.done \|\| event\?\.done\)/)
})

test('completed agent answers leave streaming text and render markdown immediately', () => {
  assert.match(agentStreamSource, /const hasTerminalAnswerEvent = \(events: any\[\]\): boolean =>/)
  assert.match(agentStreamSource, /event\?\.type === 'agent_complete'/)
  assert.match(agentStreamSource, /if \(!props\.session\?\.is_completed && !hasTerminalAnswerEvent\(events\)\) return events/)
  assert.match(agentStreamSource, /const hasActiveAnswerStream = computed\(\(\) =>/)
  assert.match(agentStreamSource, /return hasActiveAnswerStream\.value && !event\.done/)
  assert.match(agentStreamSource, /<pre v-if="isAnswerEventStreaming\(event\)" class="streaming-answer-text">/)
  assert.match(agentStreamSource, /const answerFullyRendered = computed\(\s*\(\) => isConversationDone\.value \|\| !hasActiveAnswerStream\.value,\s*\)/)
})

test('switching back to an in-flight stream anchors at the active turn start', () => {
  assert.match(source, /:data-message-index="index"/)
  assert.match(source, /const pendingInFlightTurnAnchorSessionId = ref\(''\)/)
  assert.match(source, /const activeInFlightTurnAnchorSessionId = ref\(''\)/)
  assert.match(source, /const preserveInFlightTurnScrollSessionId = ref\(''\)/)
  assert.match(source, /const clearInFlightTurnScrollPreservation = \(targetSessionId = ''\) => \{/)
  assert.match(source, /const findActiveTurnUserRenderedIndex = \(\) => \{/)
  assert.match(source, /message\?\.role !== 'assistant'[\s\S]*message\.is_completed && !message\.__stream_active/)
  assert.match(source, /const applyRenderedMessageAnchor = \(index\) => \{[\s\S]*getBoundingClientRect\(\)[\s\S]*container\.scrollTop \+ targetRect\.top - containerRect\.top - 8[\s\S]*userHasScrolledUp\.value = true/)
  assert.match(source, /const scheduleInFlightTurnAnchor = \(targetSessionId = session_id\.value\) => \{[\s\S]*window\.requestAnimationFrame/)
  assert.match(source, /if \(userIndex >= 0 && applyRenderedMessageAnchor\(userIndex\)\) \{[\s\S]*activeInFlightTurnAnchorSessionId\.value = '';[\s\S]*return;/)
  assert.match(source, /const anchorRestoredInFlightTurn = \(targetSessionId = session_id\.value\) => \{/)
  assert.match(source, /preserveInFlightTurnScrollSessionId\.value = targetSessionId[\s\S]*activeInFlightTurnAnchorSessionId\.value = targetSessionId[\s\S]*inFlightTurnAnchorUntil = Date\.now\(\) \+ 2000/)
  assert.match(source, /const keepInFlightTurnAnchor = \(targetSessionId = session_id\.value\) => \{/)
  assert.match(source, /if \(preserveInFlightTurnScrollSessionId\.value !== targetSessionId\) return;[\s\S]*if \(hasActiveStream\(targetSessionId\)\) userHasScrolledUp\.value = true;/)
  assert.match(source, /const finishInFlightTurnAnchor = \(targetSessionId = session_id\.value\) => \{/)
  assert.match(source, /const hasTargetInFlightTurn = hasActiveStream\(targetSessionId\) \|\| inFlightTurnCache\.has\(targetSessionId\)/)
  assert.match(source, /pendingInFlightTurnAnchorSessionId\.value = hasTargetInFlightTurn \? targetSessionId : ''/)
  assert.match(source, /if \(!hasTargetInFlightTurn\) clearInFlightTurnScrollPreservation\(targetSessionId\)/)
  assert.match(source, /const scrollToBottom = \(force = false\) => \{[\s\S]*if \(preserveInFlightTurnScrollSessionId\.value === session_id\.value\) return;/)
  assert.match(source, /const onClickScrollToBottom = \(\) => \{[\s\S]*clearInFlightTurnScrollPreservation\(session_id\.value\)/)
  assert.match(source, /const sendMsg = async[\s\S]*clearInFlightTurnAnchor\(\);[\s\S]*clearInFlightTurnScrollPreservation\(session_id\.value\);[\s\S]*prepareForNewOutgoingMessage\(\);/)
  assert.match(source, /if \(!isScrollType && pendingInFlightTurnAnchorSessionId\.value === targetSessionId\) \{[\s\S]*isFirstEnter\.value = false;[\s\S]*userHasScrolledUp\.value = true;/)
  assert.match(source, /onBeforeAfterMsgList: \(\) => anchorRestoredInFlightTurn\(session_id\.value\)/)
  assert.match(source, /onMessageUpdated:\s*\(message,\s*payload\) => \{[\s\S]*refreshMessageRow\(message\)[\s\S]*keepInFlightTurnAnchor\(session_id\.value\)/)
  assert.match(source, /if \(payload\?\.is_completed\) \{[\s\S]*finishInFlightTurnAnchor\(session_id\.value\)/)
  assert.match(source, /restoreCachedInFlightTurn\(targetSessionId\);[\s\S]*replayBackgroundChunks\(targetSessionId\);[\s\S]*anchorRestoredInFlightTurn\(targetSessionId\);/)
  assert.match(source, /\.chat \{[\s\S]*height: 100%;[\s\S]*overflow: hidden;/)
})

test('in-flight turn cache merges snapshots instead of overwriting with stream tails', () => {
  assert.match(source, /if \(prefix\.includes\(suffix\)\) return prefix/)
  assert.match(source, /if \(suffix\.includes\(prefix\)\) return suffix/)
  assert.match(source, /const mergeInFlightTurnCache = \(targetSessionId,\s*incomingMessages\) => \{/)
  assert.match(source, /const merged = \(inFlightTurnCache\.get\(targetSessionId\) \|\| \[\]\)\.map\(cloneMessageForInFlightCache\)/)
  assert.match(source, /mergeCachedAssistantMessage\(existing,\s*incoming\)/)
  assert.match(source, /inFlightTurnCache\.set\(targetSessionId,\s*merged\)/)
  assert.match(source, /mergeInFlightTurnCache\(targetSessionId,\s*tail\)/)
})
