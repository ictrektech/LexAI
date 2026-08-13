import { defineStore } from 'pinia'
import { ref } from 'vue'
import { archiveDocument, bulkArchiveDocuments, bulkDeleteArchiveDocuments, bulkDeleteArchiveReminders, bulkIgnoreArchiveReminderCandidates, bulkRestoreDocuments, createArchiveReminderFromCandidate, deleteArchiveDocument, getArchiveBatch, getArchiveDocument, getArchiveSettings, importArchiveFiles, listArchiveCustomers, listArchiveDocuments, listArchiveReminderCandidates, listArchiveReminders, listArchiveNotifications, restoreDocument, retryArchiveDocumentExtraction, searchArchive, streamArchiveBatch, updateArchiveDocument, updateArchiveReminder, type ArchiveBatch, type ArchiveBulkActionResult, type ArchiveCustomer, type ArchiveDocument, type ArchiveNotification, type ArchiveReminder, type ArchiveReminderCandidate, type ArchiveSearchResponse, type ArchiveSettings } from '@/api/smart-archive'

export const useSmartArchiveStore = defineStore('smartArchive', () => {
  const documents = ref<ArchiveDocument[]>([])
  const customers = ref<ArchiveCustomer[]>([])
  const reminders = ref<ArchiveReminder[]>([])
  const reminderCandidates = ref<ArchiveReminderCandidate[]>([])
  const notifications = ref<ArchiveNotification[]>([])
  const settings = ref<ArchiveSettings | null>(null)
  const current = ref<ArchiveDocument | null>(null)
  const searchResult = ref<ArchiveSearchResponse | null>(null)
  const loading = ref(false)
  const importProgress = ref(0)
  let streamController: AbortController | null = null
  let batchPollTimer: ReturnType<typeof setInterval> | null = null
  let documentListQuery = ''
  let documentListArchived = false

  async function loadDocuments(q = '', archived = false, silent = false) { documentListQuery = q; documentListArchived = archived; if (!silent) loading.value = true; try { documents.value = (await listArchiveDocuments(q, archived)).data || [] } finally { if (!silent) loading.value = false } }
  async function refreshDocuments() { try { documents.value = (await listArchiveDocuments(documentListQuery, documentListArchived)).data || [] } catch { /* A later poll or the SSE reconnect will retry. */ } }
  async function loadSettings() { settings.value = (await getArchiveSettings()).data; return settings.value }
  async function loadDocument(id: string) { current.value = (await getArchiveDocument(id)).data; return current.value }
  async function upload(files: File[]) { importProgress.value = 0; const batch = (await importArchiveFiles(files, (v) => importProgress.value = v)).data; connectBatch(batch.id); return batch }
  async function loadEntities(q = '') { customers.value = (await listArchiveCustomers(q)).data || [] }
  async function loadReminders(status = '') { reminders.value = (await listArchiveReminders(status)).data || [] }
  async function loadReminderCandidates(status = 'pending') { reminderCandidates.value = (await listArchiveReminderCandidates(status)).data || [] }
  async function createReminderFromCandidate(id: string, data: { offset_days: number; time: string; assignee_id?: string }) { const row = (await createArchiveReminderFromCandidate(id, data)).data; reminderCandidates.value = reminderCandidates.value.filter((item) => item.id !== id); reminders.value = [...reminders.value, row]; return row }
  async function loadNotifications(unread = false) { notifications.value = (await listArchiveNotifications(unread)).data || [] }
  async function search(query: string, filters: Record<string, unknown> = {}) { searchResult.value = (await searchArchive({ query, filters, page: 1, page_size: 30 })).data; return searchResult.value }
  async function updateDocument(id: string, data: Record<string, unknown>) { const row = (await updateArchiveDocument(id, data)).data; current.value = row; documents.value = documents.value.map((item) => item.id === id ? row : item); return row }
  async function retryExtraction(id: string) { const row = (await retryArchiveDocumentExtraction(id)).data; documents.value = documents.value.map((item) => item.id === id ? row : item); if (current.value?.id === id) current.value = row; return row }
  async function archive(id: string) { const row = (await archiveDocument(id)).data; documents.value = documents.value.filter((item) => item.id !== id); return row }
  async function restore(id: string) { const row = (await restoreDocument(id)).data; documents.value = documents.value.filter((item) => item.id !== id); return row }
  async function deleteDocument(id: string) { await deleteArchiveDocument(id); documents.value = documents.value.filter((item) => item.id !== id); if (current.value?.id === id) current.value = null }
  async function bulkAction(action: 'archive' | 'restore' | 'delete', ids: string[]): Promise<ArchiveBulkActionResult> {
    const response = action === 'archive' ? await bulkArchiveDocuments(ids) : action === 'restore' ? await bulkRestoreDocuments(ids) : await bulkDeleteArchiveDocuments(ids)
    const result = response.data
    const succeeded = new Set(result.items.filter((item) => item.success).map((item) => item.id))
    documents.value = documents.value.filter((item) => !succeeded.has(item.id))
    if (current.value && succeeded.has(current.value.id)) current.value = null
    return result
  }
  async function updateReminder(id: string, data: Record<string, unknown>) { const row = (await updateArchiveReminder(id, data)).data; reminders.value = reminders.value.map((item) => item.id === id ? row : item); return row }
  async function bulkDeleteReminders(ids: string[]): Promise<ArchiveBulkActionResult> { const result = (await bulkDeleteArchiveReminders(ids)).data; const succeeded = new Set(result.items.filter((item) => item.success).map((item) => item.id)); reminders.value = reminders.value.filter((item) => !succeeded.has(item.id)); return result }
  async function bulkIgnoreReminderCandidates(ids: string[]): Promise<ArchiveBulkActionResult> { const result = (await bulkIgnoreArchiveReminderCandidates(ids)).data; const succeeded = new Set(result.items.filter((item) => item.success).map((item) => item.id)); reminderCandidates.value = reminderCandidates.value.filter((item) => !succeeded.has(item.id)); return result }
  function stopBatchPolling() { if (batchPollTimer !== null) { clearInterval(batchPollTimer); batchPollTimer = null } }
  async function refreshBatch(id: string) {
    try {
      const batch = (await getArchiveBatch(id)).data
      importProgress.value = batch.total > 0 ? Math.min(100, Math.round(((batch.completed + batch.failed) / batch.total) * 100)) : 0
      await refreshDocuments()
      if (batch.status === 'completed' || batch.status === 'failed') stopBatchPolling()
    } catch { /* Keep polling; a transient request failure must not leave a stale status on screen. */ }
  }
  function connectBatch(id: string) {
    disconnect()
    streamController = new AbortController()
    void streamArchiveBatch(id, streamController.signal, (batch) => {
      importProgress.value = batch.total > 0 ? Math.min(100, Math.round(((batch.completed + batch.failed) / batch.total) * 100)) : 0
      void refreshDocuments()
      if (batch.status === 'completed' || batch.status === 'failed') stopBatchPolling()
    }).catch(() => undefined)
    // SSE gives low-latency updates, while this durable snapshot poll covers
    // proxies that buffer/close SSE and pages reopened after the upload.
    void refreshBatch(id)
    batchPollTimer = setInterval(() => { void refreshBatch(id) }, 1500)
  }
  function disconnect() { streamController?.abort(); streamController = null; stopBatchPolling() }
  return { documents, customers, reminders, reminderCandidates, notifications, settings, current, searchResult, loading, importProgress, loadDocuments, refreshDocuments, loadSettings, loadDocument, upload, loadEntities, loadReminders, loadReminderCandidates, createReminderFromCandidate, loadNotifications, search, updateDocument, retryExtraction, archive, restore, deleteDocument, bulkAction, updateReminder, bulkDeleteReminders, bulkIgnoreReminderCandidates, connectBatch, disconnect }
})
