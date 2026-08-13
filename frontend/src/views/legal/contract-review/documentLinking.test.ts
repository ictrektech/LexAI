import assert from 'node:assert/strict'
import { test } from 'node:test'
import { hasReviewQuoteMatch, normalizeReviewText } from './documentLinking.ts'

test('normalizes whitespace and smart quotes for viewer linking', () => {
  assert.equal(normalizeReviewText(' “Payment”\n terms '), '"payment"terms')
})

test('matches an exact quote despite PDF text-item whitespace', () => {
  assert.equal(hasReviewQuoteMatch('Payment shall be made within 30 days.', 'Payment shall be made\nwithin 30 days.'), true)
})

test('does not accept a short unrelated quote as a fuzzy match', () => {
  assert.equal(hasReviewQuoteMatch('Confidentiality survives termination.', 'Payment is due'), false)
})
