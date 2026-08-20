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

  test('drills into contract drafting and restores the root navigation', async ({ page }) => {
    await page.goto('/legal/contract-review')
    await page.getByTestId('legal-nav-drafting').click()

    await expect(page.getByTestId('legal-nav-back')).toBeVisible()
    await expect(page.getByTestId('legal-nav-generate-contract')).toBeVisible()
    await expect(page.getByTestId('legal-nav-edit-contract')).toBeVisible()
    await expect(page.getByTestId('legal-nav-contract-review')).toHaveCount(0)
    await expect(page.getByTestId('legal-nav-platform-console')).toHaveCount(0)

    await page.getByTestId('legal-nav-edit-contract').click()
    await expect(page).toHaveURL(/\/legal\/drafting$/)
    await expect(page.getByTestId('legal-nav-edit-contract')).toHaveAttribute('aria-current', 'page')

    await page.getByTestId('legal-nav-back').click()
    await expect(page).toHaveURL(/\/legal\/drafting$/)
    await expect(page.getByTestId('legal-nav-contract-review')).toBeVisible()
    await expect(page.getByTestId('legal-nav-drafting')).toHaveAttribute('aria-current', 'page')
    await expect(page.getByTestId('legal-nav-platform-console')).toBeVisible()
  })

  test('expands a collapsed sidebar before entering contract drafting', async ({ page }) => {
    await page.goto('/legal/contract-review')
    await page.getByTestId('legal-sidebar-collapse').click()
    await expect(page.locator('.legal-sidebar')).toHaveClass(/legal-sidebar--collapsed/)

    await page.getByTestId('legal-nav-drafting').click()
    await expect(page.locator('.legal-sidebar')).not.toHaveClass(/legal-sidebar--collapsed/)
    await expect(page.getByTestId('legal-nav-back')).toBeVisible()
    await expect(page.getByTestId('legal-nav-generate-contract')).toBeVisible()
  })

  test('opens the contract generation placeholder from a direct child route', async ({ page }) => {
    await page.goto('/legal/drafting/generate')
    await expect(page.getByTestId('contract-generation')).toBeVisible()
    await expect(page.getByTestId('legal-nav-generate-contract')).toHaveAttribute('aria-current', 'page')
    await expect(page.getByTestId('legal-nav-edit-contract')).toBeVisible()
    await expect(page.getByTestId('contract-generation')).toContainText('功能建设中')
  })

  test('opens a drafting task route and restores list filters on return', async ({ page }) => {
    await page.goto('/legal/drafting')
    await expect(page.getByTestId('legal-nav-edit-contract')).toHaveAttribute('aria-current', 'page')
    await page.getByTestId('drafting-search').fill('同名采购合同')
    await expect(page.getByTestId('drafting-row-11111111-1111-4111-8111-111111111111')).toBeVisible()
    await expect(page.getByTestId('drafting-row-22222222-2222-4222-8222-222222222222')).toBeVisible()

    await page.getByTestId('drafting-row-11111111-1111-4111-8111-111111111111').click()
    await expect(page).toHaveURL(/\/legal\/drafting\/11111111-1111-4111-8111-111111111111$/)
    await expect(page.getByTestId('legal-nav-edit-contract')).toHaveAttribute('aria-current', 'page')
    await expect(page.getByTestId('drafting-detail')).toContainText('11111111-1111-4111-8111-111111111111')
    await expect(page.getByText('#11111111')).toBeVisible()

    await page.goBack()
    await expect(page).toHaveURL(/\/legal\/drafting$/)
    await expect(page.getByTestId('drafting-search')).toHaveValue('同名采购合同')
    await expect(page.getByTestId('drafting-row-22222222-2222-4222-8222-222222222222')).toContainText('#22222222')
  })

  test('supports direct drafting detail links and lazy render previews', async ({ page }) => {
    await page.goto('/legal/drafting/11111111-1111-4111-8111-111111111111')
    await expect(page.getByTestId('drafting-detail')).toBeVisible()
    const topbarBox = await page.locator('.detail-topbar').boundingBox()
    const breadcrumbBox = await page.getByTestId('drafting-detail-breadcrumb').boundingBox()
    if (!topbarBox || !breadcrumbBox) throw new Error('drafting detail topbar is not measurable')
    expect(Math.abs((topbarBox.x + topbarBox.width / 2) - (breadcrumbBox.x + breadcrumbBox.width / 2))).toBeLessThanOrEqual(1)
    await expect(page.getByTestId('drafting-preview')).toHaveCount(0)
    await page.getByTestId('drafting-load-preview').click()
    await expect(page.getByTestId('drafting-preview')).toBeVisible()
  })

  test('opens execution diagnostics, lazily loads blobs, and shows legacy trace state', async ({ page }) => {
    await page.goto('/legal/drafting/11111111-1111-4111-8111-111111111111')
    await page.getByTestId('drafting-debug').click()
    await expect(page).toHaveURL(/\/legal\/drafting\/11111111-1111-4111-8111-111111111111\/debug$/)
    await expect(page.getByTestId('legal-nav-edit-contract')).toHaveAttribute('aria-current', 'page')
    await expect(page.getByRole('heading', { name: 'EditPlan' })).toBeVisible()
    await expect(page.locator('.table-wrap')).toContainText('付款期限为三日')

    const blobResponse = page.waitForResponse(response => response.url().includes('/debug/stages/') && response.url().includes('/blobs/inspect_text'))
    await page.getByRole('button', { name: /加载文本快照/ }).click()
    await blobResponse
    await expect(page.locator('.blob-area pre')).toContainText('付款期限为三日')

    await page.getByRole('button', { name: '对比模式' }).click()
    await expect(page.getByRole('heading', { name: '使用相同输入对比执行模式' })).toBeVisible()
    const comparisonRequest = page.waitForRequest(request => request.url().endsWith('/comparisons') && request.method() === 'POST')
    await page.getByRole('button', { name: '开始对比' }).click()
    await comparisonRequest
    await expect(page.locator('.comparison-grid')).toContainText('OfficeCLI')

    await page.goto('/legal/drafting/22222222-2222-4222-8222-222222222222/debug')
    await expect(page.locator('.legacy-notice')).toContainText('创建时未记录完整执行轨迹')
  })

  test('keeps a missing drafting task on an inline error page', async ({ page }) => {
    await page.goto('/legal/drafting/00000000-0000-4000-8000-000000000000')
    await expect(page).toHaveURL(/\/legal\/drafting\/00000000-0000-4000-8000-000000000000$/)
    await expect(page.getByText('无法加载任务详情')).toBeVisible()
    await expect(page.getByRole('button', { name: '返回任务记录' })).toBeVisible()
  })

  test('shows failed task errors expanded and refreshes after cancellation', async ({ page }) => {
    await page.goto('/legal/drafting/22222222-2222-4222-8222-222222222222')
    await expect(page.locator('.error-section')).toHaveJSProperty('open', true)
    await expect(page.getByText('文档引擎未能应用修改。')).toBeVisible()

    await page.goto('/legal/drafting/33333333-3333-4333-8333-333333333333')
    await page.getByRole('button', { name: '取消任务' }).click()
    await expect(page.getByText('已取消')).toBeVisible()
    await expect(page.getByRole('button', { name: '取消任务' })).toHaveCount(0)
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
