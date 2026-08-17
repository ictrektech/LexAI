import type { Page, Route } from '@playwright/test'

export type LegalRole = 'viewer' | 'contributor' | 'admin' | 'owner'

const now = '2026-08-17T08:00:00.000Z'

function previewPdf() {
  const content = 'BT /F1 18 Tf 72 720 Td (Payment due within 3 days) Tj ET\n'
  const objects = [
    '1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n',
    '2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n',
    '3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>\nendobj\n',
    '4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n',
    `5 0 obj\n<< /Length ${content.length} >>\nstream\n${content}endstream\nendobj\n`,
  ]
  let pdf = '%PDF-1.4\n'
  const offsets = [0]
  for (const object of objects) {
    offsets.push(pdf.length)
    pdf += object
  }
  const xrefOffset = pdf.length
  pdf += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`
  for (const offset of offsets.slice(1)) pdf += `${String(offset).padStart(10, '0')} 00000 n \n`
  pdf += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xrefOffset}\n%%EOF`
  return Buffer.from(pdf, 'ascii')
}

function review(id: string, status: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    title: id === 'review-ready' ? '待审采购合同' : '运行中的合同审查',
    title_customized: false,
    status,
    progress: status === 'completed' ? 100 : status === 'ready' ? 5 : 20,
    playbook_id: 'general-contract-review',
    playbook_version: '1.0',
    represented_party: 'neutral',
    file_name: status === 'draft' ? '' : 'purchase-contract.pdf',
    file_type: status === 'draft' ? '' : '.pdf',
    mime_type: status === 'draft' ? '' : 'application/pdf',
    file_size: status === 'draft' ? 0 : 128,
    metadata: {},
    overview: {},
    clauses: [],
    issues: [],
    created_at: now,
    updated_at: now,
    ...overrides,
  }
}

function archiveDocument(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    title: id === 'doc-1' ? '已签采购合同' : '待复核合同',
    file_name: `${id}.pdf`,
    file_type: '.pdf',
    file_size: 2048,
    document_type: 'contract',
    business_type: 'sale',
    agreement_number: 'AGR-001',
    amount: 1000,
    currency: 'CNY',
    extracted_fields: {},
    extraction_status: id === 'doc-2' ? 'needs_review' : 'completed',
    created_at: now,
    updated_at: now,
    evidence: [{
      id: `${id}-evidence`,
      document_id: id,
      field_name: 'agreement_number',
      value: 'AGR-001',
      confidence: 0.98,
      quote: '协议编号：AGR-001',
      locator_kind: 'pdf_page',
      locator: { page: 1 },
      source_start: 0,
      source_end: 12,
      is_manual: false,
    }],
    ...overrides,
  }
}

async function json(route: Route, data: unknown, status = 200) {
  await route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(data),
  })
}

async function requestBody(route: Route): Promise<Record<string, any>> {
  try {
    return route.request().postDataJSON() as Record<string, any>
  } catch {
    return {}
  }
}

export async function installLegalApi(page: Page, role: LegalRole = 'admin') {
  await page.addInitScript(({ role: initialRole }) => {
    localStorage.setItem('weknora_token', 'e2e-token')
    localStorage.setItem('weknora_selected_tenant_id', '7')
    localStorage.setItem('locale', 'zh-CN')
    localStorage.setItem('weknora_memberships', JSON.stringify([{ tenant_id: 7, tenant_name: 'E2E Workspace', role: initialRole }]))
    localStorage.setItem('weknora_user', JSON.stringify({ id: 'user-1', username: 'E2E User', email: 'e2e@example.com', tenant_id: 7 }))
    localStorage.setItem('weknora_tenant', JSON.stringify({ id: '7', name: 'E2E Workspace', owner_id: 'user-1' }))
  }, { role })

  let reviews = [
    review('review-ready', 'ready'),
    review('review-running', 'analyzing', { title: '分析中的合同', file_name: 'running.pdf' }),
  ]
  let documents = [archiveDocument('doc-1'), archiveDocument('doc-2')]
  let reminders = [{
    id: 'reminder-1', document_id: 'doc-1', assignee_id: 'user-1', type: 'contract_expiry',
    title: '合同到期提醒', description: '请在到期前确认是否续签', rule: {}, status: 'draft', confidence: 0.95, due_at: now, created_at: now,
  }]
  let candidates = [{
    id: 'candidate-1', document_id: 'doc-1', document_title: '已签采购合同', assignee_id: 'user-1', type: 'contract_expiry',
    source_field: 'expires_at', event_at: '2027-02-06T00:00:00.000Z', suggested_offset_days: 30,
    title: '合同到期提醒', description: '合同即将到期', confidence: 0.95, quote: '合同到期日：2027年2月6日', locator: {}, rule: {}, needs_review: false,
    status: 'pending', created_at: now, updated_at: now,
  }]
  let notifications = [{ id: 'notification-1', reminder_id: 'reminder-1', title: '合同到期提醒', body: '请及时处理', created_at: now }]
  let batch = { id: 'batch-1', total: 1, completed: 1, failed: 0, status: 'completed', created_at: now, updated_at: now }

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const method = request.method()
    const url = new URL(request.url())
    const path = url.pathname

    if (path === '/api/v1/auth/me' && method === 'GET') {
      await json(route, {
        success: true,
        data: {
          user: { id: 'user-1', username: 'E2E User', email: 'e2e@example.com', tenant_id: 7 },
          tenant: { id: 7, name: 'E2E Workspace', owner_id: 'user-1' },
          memberships: [{ tenant_id: 7, tenant_name: 'E2E Workspace', role }],
          capabilities: { can_create_tenant: false, auto_accept_invitation: false },
        },
      })
      return
    }
    if (path === '/api/v1/system/capabilities' && method === 'GET') {
      await json(route, { data: { edition: 'community', capabilities: {} } })
      return
    }
    if (path === '/api/v1/tenants/7/members' && method === 'GET') {
      await json(route, { success: true, data: { members: [{ user_id: 'user-1', username: 'E2E User', email: 'e2e@example.com', role, status: 'active', joined_at: now }], total: 1 } })
      return
    }

    if (path === '/api/v1/contract-review-playbooks' && method === 'GET') {
      await json(route, { success: true, data: [{ id: 'general-contract-review', name: 'General Contract Review', description: 'E2E playbook', version: '1.0' }] })
      return
    }
    if (path === '/api/v1/contract-reviews' && method === 'GET') {
      await json(route, { success: true, data: reviews.filter((item: any) => Boolean(item.archived_at) === (url.searchParams.get('archived') === 'true')) })
      return
    }
    if (path === '/api/v1/contract-reviews' && method === 'POST') {
      const created = review('review-new', 'draft', { title: '新建合同审查' })
      reviews.push(created)
      await json(route, { success: true, data: created }, 201)
      return
    }
    const reviewMatch = path.match(/^\/api\/v1\/contract-reviews\/([^/]+)(?:\/(.*))?$/)
    if (reviewMatch) {
      const id = reviewMatch[1]
      const action = reviewMatch[2] || ''
      const item: any = reviews.find((candidate: any) => candidate.id === id)
      if (action === 'document/preview' && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/pdf', body: previewPdf() })
        return
      }
      if (action === 'document' && method === 'POST') {
        if (item) Object.assign(item, { file_name: 'uploaded-contract.pdf', file_type: '.pdf', mime_type: 'application/pdf', file_size: 128, status: 'ready', progress: 5, updated_at: now })
        await json(route, { success: true, data: item })
        return
      }
      if (action === 'start' && method === 'POST') {
        const completed = review(id, 'completed', {
          title: item?.title || '新建合同审查',
          file_name: item?.file_name || 'uploaded-contract.pdf',
          file_type: '.pdf',
          overview: { overall_risk: 'high', executive_summary: '发现一项付款风险', risk_counts: { high: 1, medium: 0, low: 0 } },
          issues: [{ id: 'issue-1', review_id: id, clause_id: 'clause-1', sequence: 0, risk_level: 'high', title: '付款条件风险', explanation: '付款期限过短。', original_quote: 'Payment due within 3 days', suggestion: '建议调整为 30 日。', source_start: 0, source_end: 20 }],
        })
        reviews = reviews.map((candidate: any) => candidate.id === id ? completed : candidate)
        await json(route, { success: true, data: completed }, 202)
        return
      }
      if (action === 'retry' && method === 'POST') {
        const retried = { ...item, status: 'analyzing', progress: 20 }
        reviews = reviews.map((candidate: any) => candidate.id === id ? retried : candidate)
        await json(route, { success: true, data: retried }, 202)
        return
      }
      if (action === 'events' && method === 'GET') {
        const snapshot = JSON.stringify(item || review(id, 'completed'))
        await route.fulfill({ status: 200, headers: { 'content-type': 'text/event-stream' }, body: `event: snapshot\ndata: ${snapshot}\n\n` })
        return
      }
      if (!action && method === 'GET') {
        await json(route, { success: true, data: item || review(id, 'draft') }, item ? 200 : 404)
        return
      }
      if (!action && method === 'PATCH') {
        const body = await requestBody(route)
        if (item) Object.assign(item, body, { updated_at: now })
        await json(route, { success: true, data: item })
        return
      }
      if (!action && method === 'DELETE') {
        reviews = reviews.filter((candidate: any) => candidate.id !== id)
        await route.fulfill({ status: 204, body: '' })
        return
      }
    }
    const bulkReviewMatch = path.match(/^\/api\/v1\/contract-reviews\/bulk\/(archive|restore|delete)$/)
    if (bulkReviewMatch && method === 'POST') {
      const body = await requestBody(route)
      const ids = Array.isArray(body.ids) ? body.ids : []
      const action = bulkReviewMatch[1]
      const items = ids.map((id: string) => {
        const found: any = reviews.find((candidate: any) => candidate.id === id)
        if (!found) return { id, success: false, error: 'not found' }
        if (action === 'delete') reviews = reviews.filter((candidate: any) => candidate.id !== id)
        else found.archived_at = action === 'archive' ? now : undefined
        return { id, success: true }
      })
      await json(route, { success: true, data: { action, requested: items.length, succeeded: items.filter((item: any) => item.success).length, failed: items.filter((item: any) => !item.success).length, items } })
      return
    }

    if (path === '/api/v1/archive/settings' && method === 'GET') {
      await json(route, { success: true, data: { id: 'settings-1', managed_knowledge_base_id: 'kb-1', timezone: 'Asia/Shanghai', extraction_model_id: 'model-1', extraction_version: '1.0', trash_retention_days: 30 } })
      return
    }
    if (path === '/api/v1/archive/documents' && method === 'GET') {
      const archived = url.searchParams.get('archived') === 'true'
      const query = (url.searchParams.get('q') || '').toLowerCase()
      await json(route, { success: true, data: documents.filter((item: any) => !item.trashed_at && Boolean(item.archived_at) === archived && (!query || item.title.toLowerCase().includes(query))) })
      return
    }
    if (path === '/api/v1/archive/import-batches' && method === 'POST') {
      batch = { ...batch, status: 'completed' }
      documents.push(archiveDocument('imported-doc', { title: '导入的新合同' }))
      await json(route, { success: true, data: batch }, 202)
      return
    }
    const archiveMatch = path.match(/^\/api\/v1\/archive\/documents\/([^/]+)(?:\/(.*))?$/)
    if (archiveMatch && archiveMatch[1] !== 'bulk') {
      const id = archiveMatch[1]
      const action = archiveMatch[2] || ''
      const item: any = documents.find((candidate: any) => candidate.id === id)
      if (action === 'preview' && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/pdf', body: previewPdf() })
        return
      }
      if (action === 'archive' && method === 'POST') {
        if (item) item.archived_at = now
        await json(route, { success: true, data: item })
        return
      }
      if (action === 'restore' && method === 'POST') {
        if (item) item.archived_at = undefined
        await json(route, { success: true, data: item })
        return
      }
      if (action === 'retry-extraction' && method === 'POST') {
        if (item) Object.assign(item, { extraction_status: 'completed', error_message: undefined })
        await json(route, { success: true, data: item }, 202)
        return
      }
      if (!action && method === 'GET') {
        await json(route, { success: true, data: item }, item ? 200 : 404)
        return
      }
      if (!action && method === 'DELETE') {
        if (item) item.trashed_at = now
        await route.fulfill({ status: 204, body: '' })
        return
      }
    }
    const archiveBulk = path.match(/^\/api\/v1\/archive\/documents\/bulk\/(archive|restore|delete|purge)$/)
    if (archiveBulk && method === 'POST') {
      const body = await requestBody(route)
      const ids = Array.isArray(body.ids) ? [...new Set(body.ids)] : []
      const action = archiveBulk[1]
      const items = ids.map((id: string) => {
        const found: any = documents.find((candidate: any) => candidate.id === id)
        if (!found) return { id, success: false, error: 'not found' }
        if (action === 'archive') found.archived_at = now
        if (action === 'restore') found.archived_at = undefined
        if (action === 'delete' || action === 'purge') found.trashed_at = now
        return { id, success: true }
      })
      await json(route, { success: true, data: { action, requested: items.length, succeeded: items.filter((item: any) => item.success).length, failed: items.filter((item: any) => !item.success).length, items } })
      return
    }
    if (path === '/api/v1/archive/search' && method === 'POST') {
      await json(route, { success: true, data: { answer: '', documents, customers: [], citations: [], total: documents.length } })
      return
    }
    if (path === '/api/v1/archive/import-batches/batch-1' && method === 'GET') {
      await json(route, { success: true, data: batch })
      return
    }
    if (path === '/api/v1/archive/import-batches/batch-1/events' && method === 'GET') {
      await route.fulfill({ status: 200, headers: { 'content-type': 'text/event-stream' }, body: `event: progress\ndata: ${JSON.stringify(batch)}\n\n` })
      return
    }
    if (path === '/api/v1/archive/reminders' && method === 'GET') {
      await json(route, { success: true, data: reminders })
      return
    }
    if (path === '/api/v1/archive/reminder-candidates' && method === 'GET') {
      await json(route, { success: true, data: candidates })
      return
    }
    const candidateMatch = path.match(/^\/api\/v1\/archive\/reminder-candidates\/([^/]+)\/create$/)
    if (candidateMatch && method === 'POST') {
      const candidate: any = candidates.find((item: any) => item.id === candidateMatch[1])
      if (candidate) candidate.status = 'created'
      const created = { ...reminders[0], id: 'reminder-2', status: 'draft', title: candidate?.title || '新提醒' }
      reminders = [...reminders, created]
      await json(route, { success: true, data: created })
      return
    }
    const reminderMatch = path.match(/^\/api\/v1\/archive\/reminders\/([^/]+)$/)
    if (reminderMatch && method === 'PATCH') {
      const body = await requestBody(route)
      reminders = reminders.map((item: any) => item.id === reminderMatch[1] ? { ...item, ...body } : item)
      await json(route, { success: true, data: reminders.find((item: any) => item.id === reminderMatch[1]) })
      return
    }
    if (path === '/api/v1/archive/notifications' && method === 'GET') {
      await json(route, { success: true, data: notifications })
      return
    }
    const notificationRead = path.match(/^\/api\/v1\/archive\/notifications\/([^/]+)\/read$/)
    if (notificationRead && method === 'POST') {
      notifications = notifications.map((item: any) => item.id === notificationRead[1] ? { ...item, read_at: now } : item)
      await json(route, { success: true })
      return
    }
    const notificationDelete = path.match(/^\/api\/v1\/archive\/notifications\/([^/]+)$/)
    if (notificationDelete && method === 'DELETE') {
      notifications = notifications.filter((item: any) => item.id !== notificationDelete[1])
      await route.fulfill({ status: 204, body: '' })
      return
    }

    await json(route, { success: true, data: [] })
  })
}
