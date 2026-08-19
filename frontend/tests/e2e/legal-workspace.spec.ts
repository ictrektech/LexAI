import { test, expect } from '@playwright/test'

import { installLegalApi } from './fixtures/legalApi'

test.describe('legal workspace', () => {
  test.beforeEach(async ({ page }) => {
    await installLegalApi(page, 'admin')
  })

  test('navigates between legal tools', async ({ page }) => {
    await page.goto('/legal/contract-review')
    await expect(page.getByTestId('legal-nav-contract-review')).toHaveAttribute('aria-current', 'page')
    await page.getByTestId('legal-nav-smart-archive').click()
    await expect(page).toHaveURL(/\/legal\/smart-archive$/)
    await expect(page.getByTestId('archive-tab-documents')).toHaveClass(/active/)
  })

  test('runs a contract review from upload to risk findings', async ({ page }) => {
    await page.goto('/legal/contract-review')
    await page.getByTestId('contract-new-review').click()
    await expect(page).toHaveURL(/\/legal\/contract-review\/review-new$/)

    await page.getByTestId('contract-file-input').setInputFiles({
      name: 'purchase-contract.pdf',
      mimeType: 'application/pdf',
      buffer: Buffer.from('%PDF-1.4 e2e'),
    })
    await expect(page.getByTestId('contract-start-review')).toBeEnabled()
    await page.getByTestId('contract-start-review').click()
    await expect(page.getByTestId('contract-result-tab-issues')).toBeVisible()
    await page.getByTestId('contract-result-tab-issues').click()
    await expect(page.getByText('付款条件风险')).toBeVisible()
    await page.getByTestId('contract-issue-issue-1').click()
    await expect(page.locator('.review-text-mark')).toBeVisible()
  })

  test('imports a smart archive file and refreshes the document list', async ({ page }) => {
    await page.goto('/legal/smart-archive')
    await page.getByTestId('archive-file-input').setInputFiles({
      name: 'imported-contract.pdf',
      mimeType: 'application/pdf',
      buffer: Buffer.from('%PDF-1.4 e2e'),
    })
    await expect(page.getByTestId('archive-row-imported-doc')).toBeVisible()
  })

  test('archives and deletes a smart archive document through the detail panel', async ({ page }) => {
    await page.goto('/legal/smart-archive')
    await expect(page.getByTestId('archive-row-doc-1')).toBeVisible()

    page.once('dialog', (dialog) => dialog.accept())
    await page.getByTestId('archive-row-doc-1').click()
    await page.getByTestId('archive-archive').click()
    await expect(page.getByTestId('archive-row-doc-1')).toHaveCount(0)

    await page.getByTestId('archive-show-archived').selectOption('archived')
    await expect(page.getByTestId('archive-row-doc-1')).toBeVisible()
    page.once('dialog', (dialog) => dialog.accept())
    await page.getByTestId('archive-row-doc-1').click()
    await page.getByTestId('archive-delete').click()
    await expect(page.getByTestId('archive-row-doc-1')).toHaveCount(0)
  })

  test('filters smart archive documents and loads additional search results', async ({ page }) => {
    await page.goto('/legal/smart-archive')
    await page.getByTestId('archive-search').fill('many')
    await page.getByTestId('archive-search').press('Enter')
    await expect(page.getByTestId('archive-row-many-1')).toBeVisible()
    await expect(page.getByText('已加载 30/35')).toBeVisible()
    await page.getByRole('button', { name: '加载更多' }).click()
    await expect(page.getByTestId('archive-row-many-35')).toBeVisible()

    await page.locator('.archive-status-filter summary').click()
    await page.locator('.archive-status-filter label').filter({ hasText: '已完成' }).click()
    await expect(page.locator('.archive-document-status--completed').first()).toBeVisible()
  })

  test('creates and activates a reminder and marks its notification read', async ({ page }) => {
    await page.goto('/legal/smart-archive')
    await page.getByTestId('archive-tab-reminders').click()
    await page.getByTestId('archive-create-reminder').click()
    await expect(page.locator('.candidate-modal')).toBeVisible()
    await page.locator('.candidate-modal .primary-button').click()
    await expect(page.getByTestId('archive-activate-reminder').last()).toBeVisible()
    await page.getByTestId('archive-activate-reminder').last().click()
    await expect(page.getByTestId('archive-handle-reminder')).toBeVisible()
    await page.getByTestId('archive-mark-read').click()
    await expect(page.getByTestId('archive-mark-read')).toHaveCount(0)
  })

  test('hides smart archive mutations for a viewer', async ({ browser }) => {
    const context = await browser.newContext()
    const page = await context.newPage()
    await installLegalApi(page, 'viewer')
    await page.goto('/legal/smart-archive')
    await expect(page.getByTestId('archive-import')).toHaveCount(0)
    await page.getByTestId('archive-row-doc-1').click()
    await expect(page.getByTestId('archive-archive')).toHaveCount(0)
    await page.getByTestId('archive-tab-reminders').click()
    await expect(page.getByTestId('archive-create-reminder')).toHaveCount(0)
    await context.close()
  })

  test('shows contributor actions without admin-only archive deletion', async ({ browser }) => {
    const context = await browser.newContext()
    const page = await context.newPage()
    await installLegalApi(page, 'contributor')
    await page.goto('/legal/smart-archive')
    await expect(page.getByTestId('archive-import')).toBeVisible()
    await page.getByTestId('archive-row-doc-1').click()
    await expect(page.getByTestId('archive-archive')).toBeVisible()
    await expect(page.getByTestId('archive-delete')).toHaveCount(0)
    await page.getByTestId('archive-tab-reminders').click()
    await expect(page.getByTestId('archive-create-reminder')).toBeVisible()
    await context.close()
  })
})
