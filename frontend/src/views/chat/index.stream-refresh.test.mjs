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

const handlerSource = readFileSync(
  join(import.meta.dirname, '../../composables/useChatStreamHandler.ts'),
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
