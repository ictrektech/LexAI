import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createPinia, setActivePinia } from 'pinia'

import type { DocumentEditJob } from '@/api/document-edit'
import { useDocumentDraftingStore } from './documentDrafting'

function job(id: string, status: DocumentEditJob['status'] = 'queued'): DocumentEditJob {
  return {
    id, format: 'DOCX', mode: 'hybrid', status, file_name: `${id}.docx`, file_size: 1024,
    source_sha256: 'sha', instruction: 'Update payment terms', created_at: '2026-08-20T01:00:00Z',
    updated_at: '2026-08-20T01:00:00Z',
  }
}

test('retains drafting filters and scroll position in the session store', () => {
  setActivePinia(createPinia())
  const store = useDocumentDraftingStore()
  store.filters.query = 'payment'
  store.filters.dateFrom = '2026-08-20'
  store.setMode('adeu')
  store.toggleStatus('running')
  store.scrollTop = 420

  assert.deepEqual(store.filters, {
    query: 'payment', dateFrom: '2026-08-20', dateTo: '', modes: ['adeu'], statuses: ['running'],
  })
  assert.equal(store.scrollTop, 420)
})

test('upserts task snapshots without duplicating rows', () => {
  setActivePinia(createPinia())
  const store = useDocumentDraftingStore()
  store.upsert(job('job-1'))
  store.upsert(job('job-2'))
  store.current = job('job-1')
  store.upsert(job('job-1', 'completed'))

  assert.deepEqual(store.jobs.map((item) => item.id), ['job-2', 'job-1'])
  assert.equal(store.jobs.find((item) => item.id === 'job-1')?.status, 'completed')
  assert.equal(store.current?.status, 'completed')
})
