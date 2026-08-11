import { defineStore } from 'pinia'
import { ref } from 'vue'

import {
  createContractReview, deleteContractReview, getContractReview, listContractReviewPlaybooks, listContractReviews,
  retryContractReview, startContractReview, streamContractReview, updateContractReview,
  uploadContractReviewDocument, type ContractReview, type ReviewPlaybook,
} from '@/api/contract-review'

export const useContractReviewStore = defineStore('contractReview', () => {
  const tasks = ref<ContractReview[]>([])
  const current = ref<ContractReview | null>(null)
  const playbooks = ref<ReviewPlaybook[]>([])
  const loading = ref(false)
  const uploadProgress = ref(0)
  let streamController: AbortController | null = null

  async function loadList(archived = false) {
    loading.value = true
    try { tasks.value = (await listContractReviews(archived)).data || [] } finally { loading.value = false }
  }
  async function create() { const review = (await createContractReview()).data; current.value = review; return review }
  async function loadPlaybooks() { if (!playbooks.value.length) playbooks.value = (await listContractReviewPlaybooks()).data || [] }
  async function load(id: string) { loading.value = true; try { current.value = (await getContractReview(id)).data; return current.value } finally { loading.value = false } }
  async function update(id: string, data: Parameters<typeof updateContractReview>[1]) { const review = (await updateContractReview(id, data)).data; if (current.value?.id === id) current.value = review; return review }
  async function upload(id: string, file: File) { uploadProgress.value = 0; const review = (await uploadContractReviewDocument(id, file, (v) => uploadProgress.value = v)).data; current.value = review; connect(id); return review }
  async function start(id: string) { const review = (await startContractReview(id)).data; current.value = review; connect(id); return review }
  async function retry(id: string) { const review = (await retryContractReview(id)).data; current.value = review; connect(id); return review }
  async function remove(id: string) { await deleteContractReview(id); tasks.value = tasks.value.filter((item) => item.id !== id); if (current.value?.id === id) current.value = null }
  function connect(id: string) {
    disconnect(); streamController = new AbortController()
    void streamContractReview(id, streamController.signal, (review) => { current.value = review; if (review.status === 'completed' || review.status === 'failed' || review.status === 'ready') disconnect() }).catch(() => undefined)
  }
  function disconnect() { streamController?.abort(); streamController = null }

  return { tasks, current, playbooks, loading, uploadProgress, loadList, loadPlaybooks, create, load, update, upload, start, retry, remove, connect, disconnect }
})
