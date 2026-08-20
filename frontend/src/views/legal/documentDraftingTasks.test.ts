import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { DocumentEditJob } from '@/api/document-edit'
import { documentEditDurationMs, filterDocumentEditJobs, formatDocumentEditDuration, shortDocumentEditId, sortDocumentEditJobs } from './documentDraftingTasks'

function job(overrides: Partial<DocumentEditJob> = {}): DocumentEditJob {
  return {
    id: 'job-1', format: 'DOCX', mode: 'hybrid', status: 'completed', file_name: 'Supply Agreement.docx', file_size: 1024,
    source_sha256: 'sha', instruction: 'Change the payment term', created_at: '2026-08-19T02:00:00.000Z',
    updated_at: '2026-08-19T03:00:00.000Z', ...overrides,
  }
}

const emptyFilters = { query: '', dateFrom: '', dateTo: '', modes: [], statuses: [] }

test('filters drafting jobs by file name or instruction', () => {
  const rows = [job(), job({ id: 'job-2', file_name: 'NDA.docx', instruction: 'Add confidentiality language' })]
  assert.deepEqual(filterDocumentEditJobs(rows, { ...emptyFilters, query: 'PAYMENT' }).map(item => item.id), ['job-1'])
  assert.deepEqual(filterDocumentEditJobs(rows, { ...emptyFilters, query: 'nda' }).map(item => item.id), ['job-2'])
})

test('combines date, mode, and multi-status filters', () => {
  const rows = [
    job({ id: 'matching', mode: 'adeu', status: 'running', created_at: '2026-08-19T08:00:00' }),
    job({ id: 'wrong-mode', mode: 'hybrid', status: 'running', created_at: '2026-08-19T08:00:00' }),
    job({ id: 'wrong-status', mode: 'adeu', status: 'failed', created_at: '2026-08-19T08:00:00' }),
    job({ id: 'wrong-date', mode: 'adeu', status: 'queued', created_at: '2026-08-18T23:59:59' }),
  ]
  const result = filterDocumentEditJobs(rows, { ...emptyFilters, dateFrom: '2026-08-19', dateTo: '2026-08-19', modes: ['adeu'], statuses: ['queued', 'running'] })
  assert.deepEqual(result.map(item => item.id), ['matching'])
})

test('sorts active jobs first and then by updated time', () => {
  const rows = [
    job({ id: 'old-complete', updated_at: '2026-08-19T01:00:00Z' }),
    job({ id: 'new-complete', updated_at: '2026-08-19T04:00:00Z' }),
    job({ id: 'active', status: 'running', updated_at: '2026-08-19T02:00:00Z' }),
  ]
  assert.deepEqual(sortDocumentEditJobs(rows).map(item => item.id), ['active', 'new-complete', 'old-complete'])
})

test('calculates live and terminal durations', () => {
  assert.equal(documentEditDurationMs(job({ status: 'queued' })), null)
  const live = job({ status: 'running', started_at: '2026-08-19T02:00:00Z' })
  assert.equal(documentEditDurationMs(live, new Date('2026-08-19T02:01:05Z').getTime()), 65_000)
  const complete = job({ started_at: '2026-08-19T02:00:00Z', completed_at: '2026-08-19T03:02:03Z' })
  assert.equal(formatDocumentEditDuration(documentEditDurationMs(complete)), '1h 2m')
  assert.equal(documentEditDurationMs(job({ status: 'failed', started_at: '2026-08-19T02:00:00Z', updated_at: '2026-08-19T02:00:10Z' })), 10_000)
  assert.equal(documentEditDurationMs(job({ status: 'cancelled', started_at: '2026-08-19T02:00:00Z', updated_at: '2026-08-19T02:00:12Z' })), 12_000)
})

test('formats a compact task id for list display', () => {
  assert.equal(shortDocumentEditId('550e8400-e29b-41d4-a716-446655440000'), '550E8400')
  assert.equal(shortDocumentEditId('job-1'), 'JOB1')
})
