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

function documentEditJob(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    format: 'DOCX',
    mode: 'hybrid',
    status: 'completed',
    file_name: '同名采购合同.docx',
    file_size: 4096,
    source_sha256: `sha256-${id}`,
    instruction: '将付款期限调整为三十日',
    model_id: 'e2e-model',
    plan: { schema_version: '1.0', format: 'DOCX', base_sha256: `sha256-${id}`, apply_mode: 'atomic', operations: [{ operation_id: 'replace-payment-term', kind: 'replace_text', target: { quote: '付款期限为三日', expected_matches: 1 }, payload: { text: '付款期限为三十日' } }] },
    capabilities: { adeu: { engine_name: 'adeu', engine_version: '2.4.1', protocol_version: 'office.engine.v1' }, officecli: { engine_name: 'officecli', engine_version: '0.1.0', protocol_version: 'office.engine.v1' } },
    started_at: '2026-08-17T08:00:02.000Z',
    created_at: now,
    updated_at: '2026-08-17T08:00:08.000Z',
    completed_at: '2026-08-17T08:00:08.000Z',
    operations: [{
      id: `${id}-operation`, operation_id: 'replace-payment-term', kind: 'replace_text', part: 'word/document.xml',
      anchor_sha256: 'anchor-sha256', expected_matches: 1, actual_matches: 1, engine_name: 'adeu', engine_message: 'operation applied', status: 'applied', error_message: '', created_at: now,
      applied_at: '2026-08-17T08:00:07.000Z',
    }],
    artifacts: [
      { id: `${id}-render`, kind: 'render', file_name: 'preview.html', mime_type: 'text/html', sha256: 'render-sha256', size: 128, created_at: now },
      { id: `${id}-redline`, kind: 'redline', file_name: '同名采购合同-修订.docx', mime_type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', sha256: 'redline-sha256', size: 4096, created_at: now },
    ],
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
  const documentEditJobs = [
    documentEditJob('11111111-1111-4111-8111-111111111111'),
    documentEditJob('22222222-2222-4222-8222-222222222222', {
      status: 'failed',
      instruction: '删除自动续约条款',
      error_code: 'ENGINE_FAILED',
      error_message: '文档引擎未能应用修改。',
      artifacts: [],
      operations: [],
      updated_at: '2026-08-17T08:00:09.000Z',
      completed_at: '2026-08-17T08:00:09.000Z',
    }),
    documentEditJob('33333333-3333-4333-8333-333333333333', {
      status: 'running',
      file_name: '执行中的采购合同.docx',
      instruction: '增加验收条款',
      artifacts: [],
      operations: [],
      updated_at: '2026-08-17T08:00:05.000Z',
      completed_at: undefined,
    }),
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

    if (path === '/api/v1/document-edits/capabilities' && method === 'GET') {
      await json(route, {
        success: true,
        data: {
          capabilities: {
            adeu: { engine_name: 'adeu', engine_version: 'e2e', protocol_version: 'office.engine.v1', tracked_changes: true, comments: true, rendering: true, validation: true },
            officecli: { engine_name: 'officecli', engine_version: 'e2e', protocol_version: 'office.engine.v1', tracked_changes: false, comments: false, rendering: true, validation: true },
          },
          health: { adeu: { status: 'ok', message: '' }, officecli: { status: 'ok', message: '' } },
        },
      })
      return
    }
    if (path === '/api/v1/document-edits' && method === 'GET') {
      await json(route, { success: true, data: documentEditJobs })
      return
    }
    const documentEditMatch = path.match(/^\/api\/v1\/document-edits\/([^/]+)(?:\/(.*))?$/)
    if (documentEditMatch) {
      const id = documentEditMatch[1]
      const action = documentEditMatch[2] || ''
      const item: any = documentEditJobs.find((candidate: any) => candidate.id === id)
      if (!item) {
        await json(route, { success: false, message: '任务不存在或无权访问' }, 404)
        return
      }
      if (!action && method === 'GET') {
        await json(route, { success: true, data: item })
        return
      }
      if (action === 'debug' && method === 'GET') {
        const traceRecorded = !id.startsWith('22222222')
        await json(route, { success: true, data: {
          job: item,
          model: { id: 'e2e-model', name: 'e2e-planner', display_name: 'E2E Planner' },
          trace_recorded: traceRecorded,
          stages: traceRecorded ? [
            { id: `${id}-inspect`, job_id: id, stage: 'Inspect', attempt: 1, engine_name: 'adeu', engine_version: '2.4.1', status: 'completed', started_at: now, completed_at: now, duration_ms: 120, input_summary: {}, output_summary: { characters: 128 } },
            { id: `${id}-plan`, job_id: id, stage: 'Plan', attempt: 1, engine_name: 'model', status: 'completed', started_at: now, completed_at: now, duration_ms: 350, input_summary: { prompt_version: 'document-edit-plan-v1', temperature: 0.1, max_completion_tokens: 4096, truncated: false }, output_summary: { repair_count: 0, finish_reason: 'stop', usage: { total_tokens: 321 } } },
            { id: `${id}-apply`, job_id: id, stage: 'Apply', attempt: 1, engine_name: 'adeu', status: 'completed', started_at: now, completed_at: now, duration_ms: 440, input_summary: {}, output_summary: {} },
            { id: `${id}-validate`, job_id: id, stage: 'Validate', attempt: 1, engine_name: 'officecli', status: 'completed', started_at: now, completed_at: now, duration_ms: 90, input_summary: {}, output_summary: { warnings: 0 } },
          ] : [],
          blobs: traceRecorded ? [
            { id: `${id}-inspect-blob`, job_id: id, stage_run_id: `${id}-inspect`, kind: 'inspect_text', content_type: 'text/plain', sha256: 'inspect-sha', size: 128, created_at: now },
            { id: `${id}-planner-blob`, job_id: id, stage_run_id: `${id}-plan`, kind: 'planner_response_initial', content_type: 'application/json', sha256: 'planner-sha', size: 80, created_at: now },
            { id: `${id}-clean-blob`, job_id: id, stage_run_id: `${id}-validate`, kind: 'clean_text', content_type: 'text/plain', sha256: 'clean-sha', size: 128, created_at: now },
          ] : [],
        } })
        return
      }
      if (action.startsWith('debug/stages/') && action.includes('/blobs/') && method === 'GET') {
        const kind = decodeURIComponent(action.slice(action.lastIndexOf('/blobs/') + '/blobs/'.length))
        const body = kind === 'inspect_text' ? '付款期限为三日' : kind === 'clean_text' ? '付款期限为三十日' : '{"operations":[]}'
        await route.fulfill({ status: 200, contentType: kind.includes('text') ? 'text/plain' : 'application/json', body })
        return
      }
      if (action === 'comparison' && method === 'GET') {
        const jobs = documentEditJobs.filter((candidate: any) => candidate.id === id || (item.comparison_group_id && candidate.comparison_group_id === item.comparison_group_id))
        await json(route, { success: true, data: { group_id: item.comparison_group_id || '', jobs } })
        return
      }
      if (action === 'comparisons' && method === 'POST') {
        const body = await requestBody(route)
        item.comparison_group_id = item.comparison_group_id || 'e2e-comparison-group'
        for (const mode of body.modes || []) {
          const comparison = documentEditJob(`comparison-${mode}`, { mode, status: 'queued', comparison_group_id: item.comparison_group_id, comparison_parent_id: item.id, comparison_strategy: body.strategy, artifacts: [], operations: [], completed_at: undefined })
          documentEditJobs.push(comparison)
        }
        await json(route, { success: true, data: { group_id: item.comparison_group_id, jobs: documentEditJobs.filter((candidate: any) => candidate.comparison_group_id === item.comparison_group_id) } }, 202)
        return
      }
      if (action === 'cancel' && method === 'POST') {
        Object.assign(item, { status: 'cancelled', completed_at: now, updated_at: now })
        await json(route, { success: true })
        return
      }
      if (action === 'events' && method === 'GET') {
        await route.fulfill({ status: 200, headers: { 'content-type': 'text/event-stream' }, body: `event: snapshot\ndata: ${JSON.stringify(item)}\n\n` })
        return
      }
      if (action.startsWith('artifacts/') && method === 'GET') {
        const kind = decodeURIComponent(action.slice('artifacts/'.length))
        if (kind === 'render') {
          await route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><html><body><p>合同修订预览</p></body></html>' })
        } else {
          await route.fulfill({ status: 200, contentType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', body: Buffer.from('e2e-docx') })
        }
        return
      }
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
      const body = await requestBody(route)
      const filters = body.filters || {}
      const archived = Boolean(filters.archived)
      const query = String(body.query || '').toLowerCase()
      const source = query === 'many'
        ? Array.from({ length: 35 }, (_, index) => archiveDocument(`many-${index + 1}`, { title: `批量合同 ${index + 1}` }))
        : documents
      const matching = source.filter((item: any) => {
        if (item.trashed_at || Boolean(item.archived_at) !== archived) return false
        if (filters.document_type && item.document_type !== filters.document_type) return false
        if (Array.isArray(filters.extraction_statuses) && filters.extraction_statuses.length && !filters.extraction_statuses.includes(item.extraction_status)) return false
        if (filters.imported_from && new Date(item.created_at) < new Date(filters.imported_from)) return false
        if (filters.imported_to && new Date(item.created_at) > new Date(filters.imported_to)) return false
        return query === 'many' || !query || `${item.title} ${item.file_name} ${item.agreement_number}`.toLowerCase().includes(query)
      })
      const page = Math.max(1, Number(body.page) || 1)
      const pageSize = Math.max(1, Number(body.page_size) || 30)
      const pageRows = matching.slice((page - 1) * pageSize, page * pageSize)
      await json(route, { success: true, data: { answer: '', documents: pageRows, customers: [], citations: [], total: matching.length } })
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
