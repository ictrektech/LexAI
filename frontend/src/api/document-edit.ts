import { fetchEventSource } from '@microsoft/fetch-event-source'

import { getApiBaseUrl } from '@/utils/api-base'
import { get, post, postUpload } from '@/utils/request'

export type DocumentEditMode = 'adeu' | 'officecli' | 'hybrid'
export type DocumentEditStatus = 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface DocumentEditArtifact {
  id: string
  kind: string
  file_name: string
  mime_type: string
  sha256: string
  size: number
  created_at: string
}

export interface DocumentEditOperation {
  id: string
  operation_id: string
  kind: string
  part: string
  anchor_sha256: string
  expected_matches: number
  actual_matches?: number
  engine_name?: string
  engine_message?: string
  status: 'planned' | 'applied' | 'failed' | 'cancelled'
  error_message?: string
  created_at: string
  applied_at?: string
}

export interface DocumentEditJob {
  id: string
  format: 'DOCX'
  mode: DocumentEditMode
  status: DocumentEditStatus
  file_name: string
  file_size: number
  source_sha256: string
  instruction: string
  model_id?: string
  plan?: Record<string, any>
  capabilities?: Record<string, DocumentEngineCapability>
  comparison_group_id?: string
  comparison_parent_id?: string
  comparison_strategy?: DocumentEditComparisonStrategy
  error_code?: string
  error_message?: string
  started_at?: string
  created_at: string
  updated_at: string
  completed_at?: string
  artifacts?: DocumentEditArtifact[]
  operations?: DocumentEditOperation[]
}

export type DocumentEditComparisonStrategy = 'replan' | 'locked_plan'
export type DocumentEditStageStatus = 'running' | 'completed' | 'failed' | 'skipped'

export interface DocumentEditStageRun {
  id: string
  job_id: string
  stage: 'Inspect' | 'Plan' | 'Apply' | 'Validate' | 'Render' | 'Publish' | string
  attempt: number
  engine_name?: string
  engine_version?: string
  protocol_version?: string
  status: DocumentEditStageStatus
  started_at: string
  completed_at?: string
  duration_ms: number
  input_summary?: Record<string, any>
  output_summary?: Record<string, any>
  error_code?: string
  error_message?: string
  metadata?: Record<string, any>
}

export interface DocumentEditDebugBlob {
  id: string
  job_id: string
  stage_run_id: string
  kind: string
  content_type: string
  sha256: string
  size: number
  created_at: string
}

export interface DocumentEditDebug {
  job: DocumentEditJob
  stages: DocumentEditStageRun[]
  blobs: DocumentEditDebugBlob[]
  model?: { id: string; name: string; display_name?: string; source?: string; type?: string }
  trace_recorded: boolean
}

export interface DocumentEditComparison {
  group_id?: string
  jobs: DocumentEditJob[]
}

export interface DocumentEngineCapability {
  engine_name: string
  engine_version: string
  protocol_version: string
  format?: string
  operations?: string[]
  tracked_changes: boolean
  comments: boolean
  rendering: boolean
  validation: boolean
}

interface ApiResponse<T> { success: boolean; data: T }

export const listDocumentEdits = () => get<ApiResponse<DocumentEditJob[]>>('/api/v1/document-edits')
export const getDocumentEdit = (id: string) => get<ApiResponse<DocumentEditJob>>(`/api/v1/document-edits/${id}`)
export const cancelDocumentEdit = (id: string) => post<{ success: boolean }>(`/api/v1/document-edits/${id}/cancel`)
export const getDocumentEditDebug = (id: string) => get<ApiResponse<DocumentEditDebug>>(`/api/v1/document-edits/${id}/debug`)
export const getDocumentEditComparison = (id: string) => get<ApiResponse<DocumentEditComparison>>(`/api/v1/document-edits/${id}/comparison`)
export const createDocumentEditComparison = (id: string, modes: DocumentEditMode[], strategy: DocumentEditComparisonStrategy) =>
  post<ApiResponse<DocumentEditComparison>>(`/api/v1/document-edits/${id}/comparisons`, { modes, strategy })

export const getDocumentEditDebugBlob = (id: string, stageId: string, kind: string) =>
  get<string>(`/api/v1/document-edits/${id}/debug/stages/${stageId}/blobs/${encodeURIComponent(kind)}`, { responseType: 'text' })

export const getDocumentEditCapabilities = () =>
  get<ApiResponse<{ capabilities: Record<string, DocumentEngineCapability>; health: Record<string, { status: string; message: string }> }>>('/api/v1/document-edits/capabilities')

export function createDocumentEdit(params: {
  file: File
  instruction: string
  mode: DocumentEditMode
  modelId?: string
  editPlan?: string
  onProgress?: (value: number) => void
}) {
  const data = new FormData()
  data.append('file', params.file)
  data.append('instruction', params.instruction)
  data.append('mode', params.mode)
  if (params.modelId) data.append('model_id', params.modelId)
  if (params.editPlan) data.append('edit_plan', params.editPlan)
  return postUpload('/api/v1/document-edits', data, (event) => {
    if (event.total) params.onProgress?.(Math.round((event.loaded / event.total) * 100))
  }) as Promise<ApiResponse<DocumentEditJob>>
}

export async function downloadDocumentEditArtifact(id: string, kind: string, mimeType = ''): Promise<Blob> {
  const response = await get<ArrayBuffer>(`/api/v1/document-edits/${id}/artifacts/${encodeURIComponent(kind)}`, { responseType: 'arraybuffer' })
  return new Blob([response], mimeType ? { type: mimeType } : undefined)
}

export function streamDocumentEdit(id: string, signal: AbortSignal, onSnapshot: (job: DocumentEditJob) => void) {
  const token = localStorage.getItem('weknora_token') || ''
  const tenant = localStorage.getItem('weknora_selected_tenant_id') || ''
  return fetchEventSource(`${getApiBaseUrl()}/api/v1/document-edits/${id}/events`, {
    method: 'GET', signal, openWhenHidden: true,
    headers: {
      Authorization: `Bearer ${token}`,
      'Accept-Language': localStorage.getItem('locale') || 'zh-CN',
      ...(tenant ? { 'X-Tenant-ID': tenant } : {}),
    },
    async onopen(response) { if (!response.ok) throw new Error(`HTTP ${response.status}`) },
    onmessage(event) {
      if (event.event !== 'snapshot' || !event.data) return
      try { onSnapshot(JSON.parse(event.data) as DocumentEditJob) } catch { /* polling remains authoritative */ }
    },
    onerror() { if (!signal.aborted) return 1500 },
  })
}
