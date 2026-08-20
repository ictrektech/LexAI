<template>
  <section class="drafting-detail">
    <div v-if="loading && !job" class="detail-state"><t-loading /> {{ t('documentDrafting.loadingDetail') }}</div>
    <div v-else-if="loadError || !job" class="detail-state detail-state--error">
      <t-icon name="error-circle" size="30px" />
      <strong>{{ t('documentDrafting.detailLoadFailed') }}</strong>
      <p>{{ loadError }}</p>
      <div><button type="button" class="secondary" @click="initialize">{{ t('documentDrafting.retry') }}</button><button type="button" class="primary" @click="backToList">{{ t('documentDrafting.backToTasks') }}</button></div>
    </div>
    <template v-else>
      <header class="detail-topbar">
        <span class="detail-topbar__spacer" aria-hidden="true" />
        <div class="detail-breadcrumb" data-testid="drafting-detail-breadcrumb">
          <button data-testid="drafting-detail-back" type="button" class="breadcrumb-link" @click="backToList"><t-icon name="chevron-left" /> {{ t('documentDrafting.tasks') }}</button>
          <span>/</span>
          <code>#{{ shortDocumentEditId(job.id) }}</code>
        </div>
        <button type="button" class="copy-id" :title="t('documentDrafting.copyTaskId')" @click="copyTaskId"><t-icon name="file-copy" /> {{ t('documentDrafting.copyTaskId') }}</button>
      </header>

      <main class="detail-content" data-testid="drafting-detail">
        <section class="detail-heading">
          <div class="detail-title"><span class="file-icon"><t-icon name="file-word" /></span><div><h1>{{ job.file_name }}</h1><p>{{ t('documentDrafting.taskId') }} <code>{{ job.id }}</code></p></div></div>
          <div class="detail-actions">
            <button data-testid="drafting-debug" type="button" class="secondary" @click="openDebug"><t-icon name="system-log" /> {{ t('documentDrafting.debug.open') }}</button>
            <button v-if="isActive" type="button" class="secondary danger" :disabled="cancelling" @click="cancelTask"><t-icon name="stop-circle" /> {{ t('documentDrafting.cancel') }}</button>
            <button v-if="renderArtifact" type="button" class="secondary" :disabled="previewLoading" @click="loadPreview"><t-icon name="browse" /> {{ t('documentDrafting.preview') }}</button>
            <button v-if="primaryArtifact" type="button" class="primary" @click="download(primaryArtifact)"><t-icon name="download" /> {{ t('documentDrafting.download') }}</button>
          </div>
        </section>

        <section class="overview-card">
          <div class="preview-card">
            <div v-if="previewLoading" class="preview-state"><t-loading size="small" /> {{ t('documentDrafting.loadingPreview') }}</div>
            <iframe v-else-if="previewUrl" data-testid="drafting-preview" :src="previewUrl" sandbox="" :title="job.file_name" />
            <div v-else class="preview-state">
              <t-icon name="file-paste" size="34px" />
              <strong>{{ renderArtifact ? t('documentDrafting.previewReady') : t('documentDrafting.previewUnavailable') }}</strong>
              <p v-if="previewError">{{ previewError }}</p>
              <p v-else>{{ renderArtifact ? t('documentDrafting.previewHint') : t('documentDrafting.previewUnavailableHint') }}</p>
              <button v-if="renderArtifact" data-testid="drafting-load-preview" type="button" class="secondary" @click="loadPreview">{{ t('documentDrafting.loadPreview') }}</button>
            </div>
          </div>

          <div class="overview-info">
            <div class="overview-status">
              <span class="job-status" :class="`job-status--${job.status}`"><i />{{ t(`documentDrafting.status.${job.status}`) }}</span>
              <span class="mode-pill">{{ modeLabel(job.mode) }}</span>
              <span v-if="duration(job)" class="duration"><t-icon name="time" />{{ duration(job) }}</span>
            </div>
            <div class="instruction-block"><span>{{ t('documentDrafting.instruction') }}</span><p>{{ job.instruction }}</p></div>
            <dl class="overview-grid">
              <div><dt>{{ t('documentDrafting.created') }}</dt><dd>{{ formatFullDate(job.created_at) }}</dd></div>
              <div><dt>{{ t('documentDrafting.updated') }}</dt><dd>{{ formatFullDate(job.updated_at) }}</dd></div>
              <div><dt>{{ t('documentDrafting.fileSize') }}</dt><dd>{{ formatFileSize(job.file_size) }}</dd></div>
              <div><dt>{{ t('documentDrafting.outputs') }}</dt><dd>{{ t('documentDrafting.artifactCount', { count: documentEditArtifactCount(job) }) }}</dd></div>
              <div class="overview-grid__wide"><dt>SHA-256</dt><dd><code>{{ job.source_sha256 }}</code></dd></div>
            </dl>
          </div>
        </section>

        <section class="detail-sections">
          <details open>
            <summary><t-icon name="chevron-right" /><strong>{{ t('documentDrafting.timeline') }}</strong><span>{{ timelineSummary }}</span></summary>
            <div class="section-content"><ol class="timeline"><li><i /><div><strong>{{ t('documentDrafting.created') }}</strong><time>{{ formatFullDate(job.created_at) }}</time></div></li><li v-if="job.started_at"><i /><div><strong>{{ t('documentDrafting.started') }}</strong><time>{{ formatFullDate(job.started_at) }}</time></div></li><li v-if="job.completed_at"><i /><div><strong>{{ t('documentDrafting.completed') }}</strong><time>{{ formatFullDate(job.completed_at) }}</time></div></li></ol></div>
          </details>

          <details>
            <summary><t-icon name="chevron-right" /><strong>{{ t('documentDrafting.operations') }}</strong><span>{{ t('documentDrafting.operationCount', { count: job.operations?.length || 0 }) }}</span></summary>
            <div class="section-content"><div v-if="job.operations?.length" class="operation-list"><div v-for="operation in job.operations" :key="operation.id"><span class="operation-status" :class="`operation-status--${operation.status}`"><t-icon :name="operation.status === 'applied' ? 'check-circle' : operation.status === 'failed' ? 'error-circle' : 'time'" /></span><div><strong>{{ operation.kind }}<small v-if="operation.part">{{ operation.part }}</small></strong><p v-if="operation.error_message">{{ operation.error_message }}</p></div></div></div><p v-else class="section-empty">{{ t('documentDrafting.noOperations') }}</p></div>
          </details>

          <details>
            <summary><t-icon name="chevron-right" /><strong>{{ t('documentDrafting.outputs') }}</strong><span>{{ t('documentDrafting.artifactCount', { count: documentEditArtifactCount(job) }) }}</span></summary>
            <div class="section-content"><div v-if="job.artifacts?.length" class="artifact-list"><div v-for="artifact in job.artifacts" :key="artifact.id"><div><strong>{{ artifact.file_name }}</strong><span>{{ artifact.kind }} · {{ formatFileSize(artifact.size) }}</span></div><div><button v-if="artifact.kind === 'render'" type="button" @click="loadPreview">{{ t('documentDrafting.preview') }}</button><button type="button" @click="download(artifact)">{{ t('documentDrafting.download') }}</button></div></div></div><p v-else class="section-empty">{{ t('documentDrafting.noArtifacts') }}</p></div>
          </details>

          <details v-if="job.error_message || job.error_code" :open="job.status === 'failed'" class="error-section">
            <summary><t-icon name="chevron-right" /><strong>{{ t('documentDrafting.errorDetails') }}</strong><span>{{ job.error_code }}</span></summary>
            <div class="section-content"><code v-if="job.error_code">{{ job.error_code }}</code><p>{{ job.error_message }}</p></div>
          </details>
        </section>
      </main>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'

import { downloadDocumentEditArtifact, streamDocumentEdit, type DocumentEditArtifact, type DocumentEditJob, type DocumentEditMode } from '@/api/document-edit'
import { LEGAL_DOCUMENT_DRAFTING_DEBUG_ROUTE, LEGAL_DOCUMENT_DRAFTING_ROUTE } from '@/router/paths'
import { useDocumentDraftingStore } from '@/stores/documentDrafting'
import { copyWithToast } from '@/utils/clipboard'
import { documentEditArtifactCount, documentEditDurationMs, formatDocumentEditDuration, shortDocumentEditId } from '../documentDraftingTasks'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const store = useDocumentDraftingStore()
const loadError = ref('')
const loading = ref(true)
const cancelling = ref(false)
const previewLoading = ref(false)
const previewUrl = ref('')
const previewError = ref('')
const now = ref(Date.now())
let streamController: AbortController | null = null
let nowTimer: ReturnType<typeof setInterval> | null = null

const job = computed(() => store.current)
const isActive = computed(() => Boolean(job.value && ['queued', 'running'].includes(job.value.status)))
const renderArtifact = computed(() => job.value?.artifacts?.find((item) => item.kind === 'render') || null)
const primaryArtifact = computed(() => {
  const artifacts = job.value?.artifacts || []
  return artifacts.find((item) => item.kind === 'redline')
    || artifacts.find((item) => item.kind === 'clean')
    || artifacts.find((item) => item.kind !== 'render')
    || null
})
const timelineSummary = computed(() => job.value?.completed_at
  ? formatFullDate(job.value.completed_at)
  : job.value?.started_at ? formatFullDate(job.value.started_at) : formatFullDate(job.value?.created_at || ''))

function modeLabel(value: DocumentEditMode) { return value === 'hybrid' ? 'Hybrid' : value === 'adeu' ? 'Adeu' : 'OfficeCLI' }
function formatFullDate(value: string) { if (!value) return '—'; return new Intl.DateTimeFormat(locale.value, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(new Date(value)) }
function formatFileSize(value: number) { if (value < 1024) return `${value} B`; if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`; return `${(value / 1024 / 1024).toFixed(1)} MB` }
function duration(value: DocumentEditJob) { return formatDocumentEditDuration(documentEditDurationMs(value, now.value)) }
function backToList() { void router.push({ name: LEGAL_DOCUMENT_DRAFTING_ROUTE }) }
function openDebug() { if (job.value) void router.push({ name: LEGAL_DOCUMENT_DRAFTING_DEBUG_ROUTE, params: { jobId: job.value.id } }) }
function copyTaskId() { if (job.value) void copyWithToast(job.value.id, 'documentDrafting.taskIdCopied') }
function cleanupPreview() { if (previewUrl.value) URL.revokeObjectURL(previewUrl.value); previewUrl.value = ''; previewLoading.value = false; previewError.value = '' }
function disconnect() { streamController?.abort(); streamController = null }

function connect(value: DocumentEditJob) {
  disconnect()
  if (!['queued', 'running'].includes(value.status)) return
  streamController = new AbortController()
  void streamDocumentEdit(value.id, streamController.signal, (snapshot) => {
    store.upsert(snapshot)
    if (!['queued', 'running'].includes(snapshot.status)) disconnect()
  }).catch(() => undefined)
}

async function initialize() {
  const id = String(route.params.jobId || '')
  cleanupPreview()
  disconnect()
  loadError.value = ''
  loading.value = true
  try {
    const value = await store.loadJob(id)
    connect(value)
    if (route.query.preview === '1' && value.artifacts?.some((item) => item.kind === 'render')) await loadPreview()
  } catch (cause: any) {
    loadError.value = cause?.message || t('documentDrafting.detailLoadFailed')
  } finally {
    loading.value = false
  }
}

async function cancelTask() {
  if (!job.value) return
  cancelling.value = true
  try {
    const snapshot = await store.cancel(job.value.id)
    connect(snapshot)
  } catch (cause: any) {
    MessagePlugin.error(cause?.message || t('documentDrafting.cancelFailed'))
  } finally {
    cancelling.value = false
  }
}

async function loadPreview() {
  if (!job.value || !renderArtifact.value || previewLoading.value) return
  cleanupPreview()
  previewLoading.value = true
  try {
    const blob = await downloadDocumentEditArtifact(job.value.id, renderArtifact.value.kind, renderArtifact.value.mime_type)
    previewUrl.value = URL.createObjectURL(blob)
  } catch (cause: any) {
    previewError.value = cause?.message || t('documentDrafting.previewLoadFailed')
  } finally {
    previewLoading.value = false
  }
}

async function download(artifact: DocumentEditArtifact) {
  if (!job.value) return
  try {
    const blob = await downloadDocumentEditArtifact(job.value.id, artifact.kind, artifact.mime_type)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = artifact.file_name
    link.click()
    setTimeout(() => URL.revokeObjectURL(url), 0)
  } catch (cause: any) {
    MessagePlugin.error(cause?.message || t('documentDrafting.downloadFailed'))
  }
}

watch(() => route.params.jobId, () => { void initialize() })
onMounted(() => { void initialize(); nowTimer = setInterval(() => { now.value = Date.now() }, 1000) })
onBeforeUnmount(() => { disconnect(); cleanupPreview(); store.clearCurrent(); if (nowTimer) clearInterval(nowTimer) })
</script>

<style scoped lang="less">
.drafting-detail { width: 100%; height: 100%; overflow: auto; box-sizing: border-box; color: var(--legal-text-primary); background: var(--legal-bg-page); }
.detail-topbar { position: sticky; z-index: 5; top: 0; min-height: 50px; padding: 0 24px; display: grid; grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr); align-items: center; border-bottom: 1px solid var(--legal-border); background: var(--legal-bg-surface); color: var(--legal-text-secondary); font-size: 11px; }
.detail-breadcrumb { min-width: 0; display: inline-flex; align-items: center; justify-self: center; gap: 8px; white-space: nowrap; } .breadcrumb-link, .copy-id { display: inline-flex; align-items: center; gap: 5px; border: 0; color: var(--legal-text-secondary); background: transparent; cursor: pointer; font-size: 11px; white-space: nowrap; } .breadcrumb-link:hover, .copy-id:hover { color: var(--legal-text-primary); } .detail-topbar code { color: var(--legal-text-primary); font: 10px ui-monospace, SFMono-Regular, Menlo, monospace; } .copy-id { justify-self: end; }
.detail-content { width: min(1320px, calc(100% - 48px)); margin: 0 auto; padding: 30px 0 54px; }
.detail-heading { display: flex; align-items: center; justify-content: space-between; gap: 20px; margin-bottom: 22px; } .detail-title { min-width: 0; display: flex; align-items: center; gap: 12px; } .file-icon { width: 42px; height: 48px; display: flex; align-items: center; justify-content: center; flex: none; border: 1px solid var(--legal-border); border-radius: 6px; color: var(--legal-ai-strong); background: var(--legal-ai-soft); } .detail-title h1 { margin: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 22px; letter-spacing: -.02em; } .detail-title p { margin: 6px 0 0; color: var(--legal-text-secondary); font-size: 10px; } .detail-title code { overflow-wrap: anywhere; font: inherit; }
.detail-actions { display: flex; gap: 8px; flex: none; } .detail-actions button, .detail-state button, .preview-state button { min-height: 34px; padding: 0 12px; display: inline-flex; align-items: center; justify-content: center; gap: 6px; border-radius: 5px; cursor: pointer; font-size: 11px; font-weight: 650; } .primary { border: 1px solid var(--legal-brand); color: white; background: var(--legal-brand); } .secondary { border: 1px solid var(--legal-border); color: var(--legal-text-primary); background: var(--legal-bg-surface); } .danger { color: var(--legal-status-failed-strong); } button:disabled { opacity: .5; cursor: not-allowed; }
.overview-card { display: grid; grid-template-columns: minmax(320px, .9fr) minmax(420px, 1.35fr); border: 1px solid var(--legal-border); border-radius: 7px; background: var(--legal-bg-surface); overflow: hidden; }
.preview-card { min-height: 360px; display: flex; border-right: 1px solid var(--legal-border); background: var(--legal-bg-paper); } .preview-card iframe { width: 100%; min-height: 360px; border: 0; background: white; } .preview-state { width: 100%; padding: 30px; display: flex; flex-direction: column; align-items: center; justify-content: center; box-sizing: border-box; color: var(--legal-text-secondary); text-align: center; } .preview-state strong { margin-top: 14px; color: var(--legal-text-primary); font-size: 13px; } .preview-state p { max-width: 340px; margin: 6px 0 16px; font-size: 11px; line-height: 1.5; }
.overview-info { padding: 26px; } .overview-status { display: flex; align-items: center; gap: 12px; padding-bottom: 20px; border-bottom: 1px solid var(--legal-border); } .job-status { display: inline-flex; align-items: center; gap: 7px; font-size: 11px; font-weight: 700; } .job-status i { width: 9px; height: 9px; border-radius: 50%; background: currentColor; box-shadow: 0 0 0 3px var(--legal-status-cancelled-soft); } .job-status--queued { color: var(--legal-status-queued-strong); } .job-status--queued i { box-shadow: 0 0 0 3px var(--legal-status-queued-soft); } .job-status--running { color: var(--legal-status-running-strong); } .job-status--running i { box-shadow: 0 0 0 3px var(--legal-status-running-soft); animation: detail-status-pulse 1.8s ease-out infinite; } .job-status--completed { color: var(--legal-status-completed-strong); } .job-status--completed i { box-shadow: 0 0 0 3px var(--legal-status-completed-soft); } .job-status--failed { color: var(--legal-status-failed-strong); } .job-status--failed i { box-shadow: 0 0 0 3px var(--legal-status-failed-soft); } .job-status--cancelled { color: var(--legal-status-cancelled-strong); } .mode-pill { padding: 4px 8px; border-radius: 12px; background: var(--legal-ai-soft); font-size: 10px; } .duration { margin-left: auto; display: inline-flex; align-items: center; gap: 4px; color: var(--legal-text-secondary); font-size: 10px; }
.instruction-block { padding: 22px 0; border-bottom: 1px solid var(--legal-border); } .instruction-block span, dt { color: var(--legal-text-secondary); font-size: 10px; } .instruction-block p { margin: 8px 0 0; white-space: pre-wrap; font-size: 12px; line-height: 1.7; }
.overview-grid { margin: 18px 0 0; display: grid; grid-template-columns: 1fr 1fr; gap: 16px 24px; } .overview-grid div { min-width: 0; } .overview-grid__wide { grid-column: 1 / -1; } .overview-grid dd { margin: 5px 0 0; overflow-wrap: anywhere; font-size: 11px; } .overview-grid code { font: 10px ui-monospace, SFMono-Regular, Menlo, monospace; }
.detail-sections { margin-top: 14px; border: 1px solid var(--legal-border); border-radius: 7px; background: var(--legal-bg-surface); overflow: hidden; } .detail-sections details + details { border-top: 1px solid var(--legal-border); } .detail-sections summary { min-height: 52px; padding: 0 16px; display: grid; grid-template-columns: 18px 1fr auto; align-items: center; gap: 6px; list-style: none; cursor: pointer; } .detail-sections summary::-webkit-details-marker { display: none; } .detail-sections summary > :first-child { color: var(--legal-text-secondary); transition: transform .15s; } .detail-sections details[open] summary > :first-child { transform: rotate(90deg); } .detail-sections summary strong { font-size: 11px; } .detail-sections summary span { color: var(--legal-text-secondary); font-size: 10px; } .section-content { padding: 4px 40px 20px; }
.timeline { margin: 0; padding: 8px 0 0; list-style: none; } .timeline li { position: relative; display: flex; gap: 10px; padding: 0 0 17px; } .timeline li:not(:last-child)::after { content: ''; position: absolute; left: 3px; top: 10px; bottom: 0; width: 1px; background: var(--legal-border); } .timeline li > i { z-index: 1; width: 7px; height: 7px; margin-top: 4px; border-radius: 50%; background: var(--legal-ai-strong); } .timeline strong, .timeline time { display: block; font-size: 11px; } .timeline time { margin-top: 3px; color: var(--legal-text-secondary); }
.operation-list > div { display: flex; gap: 9px; padding: 10px 0; border-top: 1px solid var(--legal-border); } .operation-list > div:first-child { border-top: 0; } .operation-status { color: var(--legal-status-running); } .operation-status--applied { color: var(--legal-status-completed); } .operation-status--failed { color: var(--legal-status-failed); } .operation-status--cancelled { color: var(--legal-status-cancelled); } .operation-list strong { font-size: 11px; } .operation-list small { margin-left: 7px; color: var(--legal-text-secondary); font-weight: 400; } .operation-list p { margin: 4px 0 0; color: var(--legal-status-failed-strong); font-size: 10px; }
.artifact-list > div { min-height: 54px; display: flex; align-items: center; justify-content: space-between; gap: 10px; border-top: 1px solid var(--legal-border); } .artifact-list > div:first-child { border-top: 0; } .artifact-list strong, .artifact-list span { display: block; max-width: 420px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; } .artifact-list strong { font-size: 11px; } .artifact-list span { margin-top: 3px; color: var(--legal-text-secondary); font-size: 10px; } .artifact-list button { margin-left: 8px; border: 0; color: var(--legal-brand); background: transparent; cursor: pointer; font-size: 10px; }
.error-section .section-content { color: var(--legal-status-failed-strong); background: var(--legal-status-failed-soft); } .error-section code { display: block; padding-top: 14px; font-size: 10px; } .error-section p { margin: 8px 0 0; white-space: pre-wrap; font-size: 11px; line-height: 1.6; } .section-empty { margin: 10px 0 0; color: var(--legal-text-secondary); font-size: 11px; }
.detail-state { width: 100%; height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; color: var(--legal-text-secondary); } .detail-state strong { color: var(--legal-text-primary); font-size: 14px; } .detail-state p { max-width: 520px; margin: 0; text-align: center; font-size: 11px; } .detail-state div { display: flex; gap: 8px; margin-top: 6px; } .detail-state--error > :first-child { color: var(--legal-status-failed); }
@keyframes detail-status-pulse { 0%, 100% { box-shadow: 0 0 0 3px var(--legal-status-running-soft); } 50% { box-shadow: 0 0 0 6px transparent; } }
@media (max-width: 900px) { .detail-heading { align-items: flex-start; flex-direction: column; } .detail-actions { flex-wrap: wrap; } .overview-card { grid-template-columns: 1fr; } .preview-card { min-height: 300px; border-right: 0; border-bottom: 1px solid var(--legal-border); } }
@media (max-width: 620px) { .detail-topbar { padding: 0 14px; } .copy-id { width: 28px; padding: 0; justify-content: center; overflow: hidden; white-space: nowrap; } .copy-id :deep(.t-icon) { flex: none; } .detail-content { width: calc(100% - 28px); padding-top: 20px; } .detail-title h1 { font-size: 18px; } .detail-title p { max-width: 250px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; } .detail-actions { width: 100%; } .detail-actions button { flex: 1; } .overview-info { padding: 20px; } .overview-grid { grid-template-columns: 1fr; } .section-content { padding: 4px 20px 18px; } .detail-sections summary { padding: 0 12px; } .artifact-list > div { align-items: flex-start; flex-direction: column; padding: 12px 0; } }

/* Readability overrides: keep the detail page scannable at normal zoom. */
.detail-topbar { min-height: 56px; font-size: 12px; }
.breadcrumb-link, .copy-id { font-size: 12px; }
.detail-topbar code { font-size: 11px; }
.detail-title p { font-size: 12px; }
.detail-actions button, .detail-state button, .preview-state button { min-height: 38px; font-size: 13px; }
.preview-state p { font-size: 12px; line-height: 1.6; }
.job-status { font-size: 12px; }
.mode-pill { font-size: 11px; }
.duration { font-size: 12px; }
.instruction-block span, dt { font-size: 12px; }
.instruction-block p { font-size: 14px; line-height: 1.75; }
.overview-grid dd { font-size: 13px; }
.overview-grid code { font-size: 11px; }
.detail-sections summary { min-height: 58px; padding-left: 18px; padding-right: 18px; }
.detail-sections summary strong { font-size: 13px; }
.detail-sections summary span { font-size: 12px; }
.section-content { padding-top: 8px; padding-bottom: 24px; }
.timeline strong { font-size: 13px; }
.timeline time { font-size: 12px; }
.operation-list strong { font-size: 13px; }
.operation-list small { font-size: 12px; }
.operation-list p { font-size: 12px; }
.artifact-list strong { font-size: 13px; }
.artifact-list span, .artifact-list button { font-size: 12px; }
.error-section code { font-size: 11px; }
.error-section p { font-size: 13px; }
.section-empty, .detail-state p { font-size: 12px; }
</style>
