import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'

import {
  cancelDocumentEdit,
  createDocumentEdit,
  getDocumentEdit,
  getDocumentEditCapabilities,
  listDocumentEdits,
  type DocumentEditJob,
  type DocumentEditMode,
  type DocumentEditStatus,
  type DocumentEngineCapability,
} from '@/api/document-edit'
import type { DocumentDraftingFilters } from '@/views/legal/documentDraftingTasks'

export const useDocumentDraftingStore = defineStore('documentDrafting', () => {
  const jobs = ref<DocumentEditJob[]>([])
  const current = ref<DocumentEditJob | null>(null)
  const capabilities = ref<Record<string, DocumentEngineCapability>>({})
  const health = ref<Record<string, { status: string; message: string }>>({})
  const loadingJobs = ref(false)
  const loadingCurrent = ref(false)
  const loadingCapabilities = ref(false)
  const initialized = ref(false)
  const scrollTop = ref(0)
  const filters = reactive<DocumentDraftingFilters>({
    query: '',
    dateFrom: '',
    dateTo: '',
    modes: [],
    statuses: [],
  })

  function upsert(job: DocumentEditJob) {
    const index = jobs.value.findIndex((item) => item.id === job.id)
    if (index >= 0) jobs.value[index] = job
    else jobs.value.unshift(job)
    if (current.value?.id === job.id) current.value = job
    return job
  }

  async function loadList(silent = false) {
    if (!silent) loadingJobs.value = true
    try {
      jobs.value = (await listDocumentEdits()).data || []
      initialized.value = true
      return jobs.value
    } finally {
      if (!silent) loadingJobs.value = false
    }
  }

  async function loadJob(id: string) {
    if (current.value?.id !== id) current.value = null
    loadingCurrent.value = true
    try {
      const job = (await getDocumentEdit(id)).data
      current.value = job
      upsert(job)
      return job
    } finally {
      loadingCurrent.value = false
    }
  }

  async function loadCapabilities() {
    loadingCapabilities.value = true
    try {
      const response = (await getDocumentEditCapabilities()).data
      capabilities.value = response.capabilities || {}
      health.value = response.health || {}
      return response
    } finally {
      loadingCapabilities.value = false
    }
  }

  async function create(params: { file: File; instruction: string; mode: DocumentEditMode }) {
    const job = (await createDocumentEdit(params)).data
    return upsert(job)
  }

  async function cancel(id: string) {
    await cancelDocumentEdit(id)
    return loadJob(id)
  }

  function toggleStatus(status: DocumentEditStatus) {
    filters.statuses = filters.statuses.includes(status)
      ? filters.statuses.filter((item) => item !== status)
      : [...filters.statuses, status]
  }

  function setMode(mode: DocumentEditMode | '') {
    filters.modes = mode ? [mode] : []
  }

  function clearFilters() {
    filters.query = ''
    filters.dateFrom = ''
    filters.dateTo = ''
    filters.modes = []
    filters.statuses = []
  }

  function clearCurrent() { current.value = null }

  return {
    jobs, current, capabilities, health, loadingJobs, loadingCurrent, loadingCapabilities,
    initialized, scrollTop, filters, upsert, loadList, loadJob, loadCapabilities, create,
    cancel, toggleStatus, setMode, clearFilters, clearCurrent,
  }
})
