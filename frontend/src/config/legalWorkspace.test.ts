import assert from 'node:assert/strict'
import { test } from 'node:test'

import {
  LEGAL_WORKSPACE_NAV_ITEMS,
  isLegalWorkspaceItemActive,
  legalWorkspaceNavPath,
  legalWorkspaceItemsFor,
} from './legalWorkspace.ts'
import {
  LEGAL_ASSISTANT_CHAT_ROUTE,
  LEGAL_ASSISTANT_HOME_ROUTE,
  LEGAL_CONTRACT_GENERATION_ROUTE,
  LEGAL_CONTRACT_REVIEW_ROUTE,
  LEGAL_CONTRACT_REVIEW_DETAIL_ROUTE,
  LEGAL_DOCUMENT_DRAFTING_DETAIL_ROUTE,
  LEGAL_DOCUMENT_DRAFTING_DEBUG_ROUTE,
  LEGAL_DOCUMENT_DRAFTING_ROUTE,
} from '../router/paths.ts'

test('legal workspace navigation keeps enabled tools and future placeholders distinct', () => {
  assert.deepEqual(legalWorkspaceItemsFor('primary'), [])

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
    ['legal-research'],
  )
  assert.ok(tools.filter((item) => !item.disabled && !item.children?.length).every((item) => item.destination))

  const drafting = tools.find((item) => item.id === 'drafting')!
  assert.equal(drafting.destination, undefined)
  assert.deepEqual(drafting.children?.map((item) => item.id), ['generate-contract', 'edit-contract'])
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

test('drafting navigation remains active on list and detail routes', () => {
  const drafting = LEGAL_WORKSPACE_NAV_ITEMS.find((item) => item.id === 'drafting')!

  assert.equal(isLegalWorkspaceItemActive(drafting, LEGAL_DOCUMENT_DRAFTING_ROUTE), true)
  assert.equal(isLegalWorkspaceItemActive(drafting, LEGAL_DOCUMENT_DRAFTING_DETAIL_ROUTE), true)
  assert.equal(isLegalWorkspaceItemActive(drafting, LEGAL_DOCUMENT_DRAFTING_DEBUG_ROUTE), true)
  assert.equal(isLegalWorkspaceItemActive(drafting, LEGAL_CONTRACT_REVIEW_ROUTE), false)
})

test('drafting navigation resolves child paths for direct links', () => {
  for (const routeName of [
    LEGAL_DOCUMENT_DRAFTING_ROUTE,
    LEGAL_DOCUMENT_DRAFTING_DETAIL_ROUTE,
    LEGAL_DOCUMENT_DRAFTING_DEBUG_ROUTE,
  ]) {
    assert.deepEqual(legalWorkspaceNavPath(routeName).map((item) => item.id), [
      'drafting',
      'edit-contract',
    ])
  }
  assert.deepEqual(legalWorkspaceNavPath(LEGAL_CONTRACT_GENERATION_ROUTE).map((item) => item.id), [
    'drafting',
    'generate-contract',
  ])
  assert.equal(isLegalWorkspaceItemActive(
    LEGAL_WORKSPACE_NAV_ITEMS.find((item) => item.id === 'drafting')!,
    LEGAL_CONTRACT_GENERATION_ROUTE,
  ), true)
})
