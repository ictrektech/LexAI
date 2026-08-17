import test from 'node:test'
import assert from 'node:assert/strict'

import {
  archiveDocumentIsProcessing,
  archiveImportProgress,
  contractReviewIsSettled,
  removeSuccessfulRows,
} from './legalState'

test('bulk state removes only successful rows and retains failures', () => {
  const rows = [{ id: 'ok' }, { id: 'failed' }, { id: 'untouched' }]
  const result = {
    items: [
      { id: 'ok', success: true },
      { id: 'failed', success: false, error: 'not found' },
    ],
  }

  assert.deepEqual(removeSuccessfulRows(rows, result), [{ id: 'failed' }, { id: 'untouched' }])
})

test('contract review polling stops only at durable terminal states', () => {
  assert.equal(contractReviewIsSettled('ready'), true)
  assert.equal(contractReviewIsSettled('completed'), true)
  assert.equal(contractReviewIsSettled('failed'), true)
  assert.equal(contractReviewIsSettled('analyzing'), false)
  assert.equal(contractReviewIsSettled('reviewing_clauses'), false)
})

test('archive processing states keep the fallback poll active', () => {
  assert.equal(archiveDocumentIsProcessing('uploading'), true)
  assert.equal(archiveDocumentIsProcessing('parsing'), true)
  assert.equal(archiveDocumentIsProcessing('extracting'), true)
  assert.equal(archiveDocumentIsProcessing('linking'), true)
  assert.equal(archiveDocumentIsProcessing('completed'), false)
  assert.equal(archiveDocumentIsProcessing('failed'), false)
})

test('archive import progress combines completed and failed files and caps at 100', () => {
  assert.equal(archiveImportProgress({ total: 4, completed: 1, failed: 1 }), 50)
  assert.equal(archiveImportProgress({ total: 4, completed: 4, failed: 1 }), 100)
  assert.equal(archiveImportProgress({ total: 0, completed: 0, failed: 0 }), 0)
})
