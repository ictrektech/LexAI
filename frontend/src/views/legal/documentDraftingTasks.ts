import type { DocumentEditJob, DocumentEditMode, DocumentEditStatus } from '@/api/document-edit'

export interface DocumentDraftingFilters {
  query: string
  dateFrom: string
  dateTo: string
  modes: DocumentEditMode[]
  statuses: DocumentEditStatus[]
}

const ACTIVE_STATUSES = new Set<DocumentEditStatus>(['queued', 'running'])

function startOfLocalDay(value: string): number | null {
  if (!value) return null
  const timestamp = new Date(`${value}T00:00:00`).getTime()
  return Number.isNaN(timestamp) ? null : timestamp
}

function endOfLocalDay(value: string): number | null {
  if (!value) return null
  const timestamp = new Date(`${value}T23:59:59.999`).getTime()
  return Number.isNaN(timestamp) ? null : timestamp
}

export function filterDocumentEditJobs(jobs: DocumentEditJob[], filters: DocumentDraftingFilters) {
  const query = filters.query.trim().toLocaleLowerCase()
  const dateFrom = startOfLocalDay(filters.dateFrom)
  const dateTo = endOfLocalDay(filters.dateTo)
  const modes = new Set(filters.modes)
  const statuses = new Set(filters.statuses)

  return jobs.filter((job) => {
    const createdAt = new Date(job.created_at).getTime()
    if (query && !`${job.file_name}\n${job.instruction}`.toLocaleLowerCase().includes(query)) return false
    if (dateFrom !== null && createdAt < dateFrom) return false
    if (dateTo !== null && createdAt > dateTo) return false
    if (modes.size && !modes.has(job.mode)) return false
    if (statuses.size && !statuses.has(job.status)) return false
    return true
  })
}

export function sortDocumentEditJobs(jobs: DocumentEditJob[]) {
  return [...jobs].sort((left, right) => {
    const activeDifference = Number(ACTIVE_STATUSES.has(right.status)) - Number(ACTIVE_STATUSES.has(left.status))
    if (activeDifference) return activeDifference
    return new Date(right.updated_at).getTime() - new Date(left.updated_at).getTime()
  })
}

export function documentEditDurationMs(job: DocumentEditJob, now = Date.now()): number | null {
  if (!job.started_at) return null
  const startedAt = new Date(job.started_at).getTime()
  if (Number.isNaN(startedAt)) return null

  let endedAt = now
  if (job.status !== 'running') {
    endedAt = new Date(job.completed_at || job.updated_at).getTime()
  }
  if (Number.isNaN(endedAt) || endedAt < startedAt) return null
  return endedAt - startedAt
}

export function formatDocumentEditDuration(durationMs: number | null): string {
  if (durationMs === null) return ''
  const seconds = Math.max(0, Math.floor(durationMs / 1000))
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`
  const hours = Math.floor(minutes / 60)
  return `${hours}h ${minutes % 60}m`
}

export function documentEditArtifactCount(job: DocumentEditJob): number {
  return job.artifacts?.length || 0
}

export function shortDocumentEditId(id: string, length = 8): string {
  const normalized = id.replaceAll('-', '')
  return normalized.slice(0, length).toUpperCase()
}
