import assert from 'node:assert/strict'
import { test } from 'node:test'

import {
  LEGAL_WORKSPACE_NAV_ITEMS,
  isLegalWorkspaceItemActive,
  legalWorkspaceItemsFor,
} from './legalWorkspace.ts'
import {
  LEGAL_ASSISTANT_CHAT_ROUTE,
  LEGAL_ASSISTANT_HOME_ROUTE,
  LEGAL_CONTRACT_REVIEW_ROUTE,
  LEGAL_CONTRACT_REVIEW_DETAIL_ROUTE,
} from '../router/paths.ts'

test('legal workspace navigation keeps enabled tools and future placeholders distinct', () => {
  const tools = legalWorkspaceItemsFor('tools')

  assert.deepEqual(tools.map((item) => item.id), [
    'ai-assistant',
    'contract-review',
    'smart-archive',
    'legal-research',
    'drafting',
  ])
  assert.deepEqual(
    tools.filter((item) => item.disabled).map((item) => item.id),
    ['legal-research', 'drafting'],
  )
  assert.ok(tools.filter((item) => !item.disabled).every((item) => item.destination))
})

test('assistant and contract routes activate only their matching tool', () => {
  const assistant = LEGAL_WORKSPACE_NAV_ITEMS.find((item) => item.id === 'ai-assistant')!
  const contract = LEGAL_WORKSPACE_NAV_ITEMS.find((item) => item.id === 'contract-review')!

  assert.equal(isLegalWorkspaceItemActive(assistant, LEGAL_ASSISTANT_HOME_ROUTE), true)
  assert.equal(isLegalWorkspaceItemActive(assistant, LEGAL_ASSISTANT_CHAT_ROUTE), true)
  assert.equal(isLegalWorkspaceItemActive(assistant, LEGAL_CONTRACT_REVIEW_ROUTE), false)
  assert.equal(isLegalWorkspaceItemActive(contract, LEGAL_CONTRACT_REVIEW_ROUTE), true)
  assert.equal(isLegalWorkspaceItemActive(contract, LEGAL_CONTRACT_REVIEW_DETAIL_ROUTE), true)
  assert.equal(isLegalWorkspaceItemActive(contract, LEGAL_ASSISTANT_CHAT_ROUTE), false)
})
