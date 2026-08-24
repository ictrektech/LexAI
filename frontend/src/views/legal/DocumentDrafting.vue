<template>
  <section ref="draftingRoot" class="drafting">
    <header class="drafting__header">
      <div>
        <span class="eyebrow">{{ t('documentDrafting.eyebrow') }}</span>
        <h1>{{ t('legalWorkspace.editContract') }}</h1>
        <p>{{ t('documentDrafting.subtitle') }}</p>
      </div>
      <button type="button" class="secondary" :disabled="store.loadingCapabilities" @click="loadCapabilities">{{ t('documentDrafting.refresh') }}</button>
    </header>

    <div class="drafting__grid">
      <form class="panel form" @submit.prevent="submit">
        <h2>{{ t('documentDrafting.newTask') }}</h2>
        <label>{{ t('documentDrafting.file') }}<input ref="fileInput" type="file" accept=".docx,application/vnd.openxmlformats-officedocument.wordprocessingml.document" @change="onFile" /></label>
        <label>{{ t('documentDrafting.instruction') }}<textarea v-model.trim="instruction" rows="7" :placeholder="t('documentDrafting.instructionPlaceholder')" /></label>
        <label>{{ t('documentDrafting.mode') }}<select v-model="mode"><option value="hybrid" :disabled="store.health.adeu?.status !== 'ok' || store.health.officecli?.status !== 'ok'">Hybrid</option><option value="adeu" :disabled="store.health.adeu?.status !== 'ok'">Adeu · Track Changes</option><option value="officecli" :disabled="store.health.officecli?.status !== 'ok'">OfficeCLI · clean-only</option></select></label>
        <p class="hint">{{ mode === 'officecli' ? t('documentDrafting.officeCliLimit') : t('documentDrafting.modeHint') }}</p>
        <button type="submit" class="primary" :disabled="submitting || !selectedFile || !instruction || !modeReady">{{ submitting ? t('documentDrafting.submitting') : t('legalWorkspace.editContract') }}</button>
        <p v-if="error" class="error">{{ error }}</p>
      </form>

      <div class="panel status">
        <h2>{{ t('documentDrafting.capabilities') }}</h2>
        <div v-for="(item, name) in store.capabilities" :key="name" class="engine-row">
          <div><strong>{{ name }}</strong><span>{{ item.protocol_version || 'office.engine.v1' }}</span></div>
          <b :class="store.health[name]?.status === 'ok' ? 'ok' : 'muted'">{{ store.health[name]?.status || 'unknown' }}</b>
        </div>
        <p class="hint">{{ t('documentDrafting.capabilityHint') }}</p>
      </div>
    </div>

    <section class="jobs">
      <div class="jobs__title">
        <div><h2>{{ t('documentDrafting.tasks') }}</h2><span>{{ t('documentDrafting.taskCount', { count: filteredJobs.length }) }}</span></div>
        <button type="button" class="secondary" :disabled="store.loadingJobs" @click="loadJobs(false)">{{ t('documentDrafting.refresh') }}</button>
      </div>

      <div class="jobs__filters">
        <label class="filter filter--search">
          <t-icon name="search" />
          <input v-model.trim="store.filters.query" data-testid="drafting-search" type="search" :placeholder="t('documentDrafting.searchPlaceholder')" :aria-label="t('documentDrafting.searchPlaceholder')" />
        </label>
        <div class="date-filter">
          <t-icon name="calendar" />
          <input v-model="store.filters.dateFrom" type="date" :aria-label="t('documentDrafting.dateFrom')" />
          <span>–</span>
          <input v-model="store.filters.dateTo" type="date" :aria-label="t('documentDrafting.dateTo')" />
        </div>
        <label class="filter filter--select">
          <select v-model="modeFilter" :aria-label="t('documentDrafting.modeFilter')">
            <option value="">{{ t('documentDrafting.allModes') }}</option>
            <option value="hybrid">Hybrid</option>
            <option value="adeu">Adeu</option>
            <option value="officecli">OfficeCLI</option>
          </select>
        </label>
        <details class="status-filter">
          <summary><span class="status-dots"><i v-for="status in allStatuses" :key="status" :class="`dot--${status}`" /></span>{{ statusFilterLabel }}<t-icon name="chevron-down" /></summary>
          <div class="status-filter__menu">
            <label v-for="status in allStatuses" :key="status"><input type="checkbox" :checked="store.filters.statuses.includes(status)" @change="store.toggleStatus(status)" /><i :class="`dot--${status}`" />{{ t(`documentDrafting.status.${status}`) }}</label>
          </div>
        </details>
        <button v-if="hasFilters" type="button" class="clear-filters" @click="store.clearFilters">{{ t('documentDrafting.clearFilters') }}</button>
      </div>

      <div class="jobs__table">
        <div class="jobs__head"><span>{{ t('documentDrafting.document') }}</span><span>{{ t('documentDrafting.statusLabel') }}</span><span>{{ t('documentDrafting.mode') }}</span><span>{{ t('documentDrafting.outputs') }}</span><span>{{ t('documentDrafting.updated') }}</span><span /></div>
        <div v-if="store.loadingJobs && !store.jobs.length" class="jobs__empty"><t-loading size="small" /> {{ t('documentDrafting.loadingTasks') }}</div>
        <div v-else-if="jobsError" class="jobs__empty jobs__empty--error"><t-icon name="error-circle" size="24px" /><strong>{{ jobsError }}</strong><button type="button" class="secondary" @click="loadJobs(false)">{{ t('documentDrafting.retry') }}</button></div>
        <div v-else-if="!store.jobs.length" class="jobs__empty"><t-icon name="file-paste" size="28px" /><strong>{{ t('documentDrafting.empty') }}</strong><p>{{ t('documentDrafting.emptyDescription') }}</p></div>
        <div v-else-if="!filteredJobs.length" class="jobs__empty"><t-icon name="search" size="26px" /><strong>{{ t('documentDrafting.noMatches') }}</strong><button type="button" class="secondary" @click="store.clearFilters">{{ t('documentDrafting.clearFilters') }}</button></div>
        <template v-else>
          <article v-for="job in filteredJobs" :key="job.id" :data-testid="`drafting-row-${job.id}`" class="job">
            <RouterLink class="job__link" :to="jobRoute(job.id)" :aria-label="`${job.file_name} · ${t('documentDrafting.taskId')} ${shortDocumentEditId(job.id)}`" />
            <div class="job__main"><i><t-icon name="file-word" /></i><div><strong>{{ job.file_name }}</strong><span class="job__instruction">{{ job.instruction }}</span><code class="job__id" :title="job.id">{{ t('documentDrafting.taskId') }} #{{ shortDocumentEditId(job.id) }}</code></div></div>
            <div class="job__status" :class="`job__status--${job.status}`"><span class="status-indicator" :class="`status-indicator--${job.status}`" /><b>{{ t(`documentDrafting.status.${job.status}`) }}</b></div>
            <span class="mode-pill">{{ modeLabel(job.mode) }}</span>
            <span class="job__artifacts">{{ t('documentDrafting.artifactCount', { count: documentEditArtifactCount(job) }) }}</span>
            <time :datetime="job.updated_at">{{ formatDate(job.updated_at) }}</time>
            <div class="job__actions" @click.stop>
              <button v-if="job.status === 'queued' || job.status === 'running'" type="button" :title="t('documentDrafting.cancel')" @click="cancel(job)"><t-icon name="stop-circle" /></button>
              <button v-if="job.status === 'completed' && hasArtifact(job, 'render')" type="button" :title="t('documentDrafting.preview')" @click="openJob(job.id, true)"><t-icon name="browse" /></button>
              <button type="button" :title="t('documentDrafting.details')" @click="openJob(job.id)"><t-icon name="chevron-right" /></button>
            </div>
          </article>
        </template>
      </div>
    </section>

  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import type { DocumentEditJob, DocumentEditMode, DocumentEditStatus } from '@/api/document-edit'
import { streamDocumentEdit } from '@/api/document-edit'
import { LEGAL_DOCUMENT_DRAFTING_DETAIL_ROUTE } from '@/router/paths'
import { useDocumentDraftingStore } from '@/stores/documentDrafting'
import { documentEditArtifactCount, filterDocumentEditJobs, shortDocumentEditId, sortDocumentEditJobs } from './documentDraftingTasks'

const { t, locale } = useI18n()
const router = useRouter()
const store = useDocumentDraftingStore()
const draftingRoot = ref<HTMLElement | null>(null)
const fileInput = ref<HTMLInputElement>()
const selectedFile = ref<File | null>(null)
const instruction = ref('')
const mode = ref<DocumentEditMode>('hybrid')
const submitting = ref(false)
const error = ref('')
const jobsError = ref('')
const streams = new Map<string, AbortController>()
const allStatuses: DocumentEditStatus[] = ['queued', 'running', 'completed', 'failed', 'cancelled']

const modeFilter = computed({
  get: () => store.filters.modes[0] || '',
  set: (value: DocumentEditMode | '') => store.setMode(value),
})
const modeReady = computed(() => {
  const required = mode.value === 'hybrid' ? ['adeu', 'officecli'] : [mode.value]
  return required.every((name) => store.health[name]?.status === 'ok')
})
const filteredJobs = computed(() => sortDocumentEditJobs(filterDocumentEditJobs(store.jobs, store.filters)))
const hasFilters = computed(() => Boolean(store.filters.query || store.filters.dateFrom || store.filters.dateTo || store.filters.modes.length || store.filters.statuses.length))
const statusFilterLabel = computed(() => store.filters.statuses.length ? t('documentDrafting.selectedStatuses', { count: store.filters.statuses.length }) : t('documentDrafting.allStatuses'))

function onFile(event: Event) { selectedFile.value = (event.target as HTMLInputElement).files?.[0] || null }
function hasArtifact(job: DocumentEditJob, kind: string) { return (job.artifacts || []).some((item) => item.kind === kind) }
function modeLabel(value: DocumentEditMode) { return value === 'hybrid' ? 'Hybrid' : value === 'adeu' ? 'Adeu' : 'OfficeCLI' }
function formatDate(value: string) { return new Intl.DateTimeFormat(locale.value, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }).format(new Date(value)) }
function jobRoute(id: string) { return { name: LEGAL_DOCUMENT_DRAFTING_DETAIL_ROUTE, params: { jobId: id } } }
function openJob(id: string, preview = false) { void router.push({ ...jobRoute(id), query: preview ? { preview: '1' } : {} }) }

async function loadCapabilities() {
  error.value = ''
  try { await store.loadCapabilities() } catch (cause: any) { error.value = cause?.message || t('documentDrafting.loadFailed') }
}
async function loadJobs(silent = false) {
  if (!silent) jobsError.value = ''
  try {
    await store.loadList(silent)
    reconcileStreams()
  } catch (cause: any) {
    if (!silent || !store.jobs.length) jobsError.value = cause?.message || t('documentDrafting.loadFailed')
  }
}
function reconcileStreams() {
  const snapshots = new Map(store.jobs.map((job) => [job.id, job]))
  streams.forEach((controller, id) => {
    const snapshot = snapshots.get(id)
    if (!snapshot || ['completed', 'failed', 'cancelled'].includes(snapshot.status)) {
      controller.abort()
      streams.delete(id)
    }
  })
  store.jobs.forEach(watchJob)
}
function watchJob(job: DocumentEditJob) {
  if (['completed', 'failed', 'cancelled'].includes(job.status) || streams.has(job.id)) return
  const controller = new AbortController()
  streams.set(job.id, controller)
  void streamDocumentEdit(job.id, controller.signal, (next) => {
    store.upsert(next)
    if (['completed', 'failed', 'cancelled'].includes(next.status)) {
      controller.abort()
      streams.delete(job.id)
    }
  }).catch(() => undefined)
}
async function submit() {
  if (!selectedFile.value || !instruction.value) return
  submitting.value = true
  error.value = ''
  try {
    const job = await store.create({ file: selectedFile.value, instruction: instruction.value, mode: mode.value })
    watchJob(job)
    selectedFile.value = null
    instruction.value = ''
    if (fileInput.value) fileInput.value.value = ''
  } catch (cause: any) {
    error.value = cause?.message || t('documentDrafting.submitFailed')
  } finally {
    submitting.value = false
  }
}
async function cancel(job: DocumentEditJob) {
  try {
    streams.get(job.id)?.abort()
    streams.delete(job.id)
    const snapshot = await store.cancel(job.id)
    watchJob(snapshot)
  } catch (cause: any) {
    watchJob(job)
    MessagePlugin.error(cause?.message || t('documentDrafting.cancelFailed'))
  }
}

onMounted(async () => {
  void loadCapabilities()
  if (store.initialized) {
    store.jobs.forEach(watchJob)
    await nextTick()
    if (draftingRoot.value) draftingRoot.value.scrollTop = store.scrollTop
    void loadJobs(true)
    return
  }
  await loadJobs(false)
  await nextTick()
  if (draftingRoot.value) draftingRoot.value.scrollTop = store.scrollTop
})
onBeforeUnmount(() => {
  store.scrollTop = draftingRoot.value?.scrollTop || 0
  streams.forEach((controller) => controller.abort())
  streams.clear()
})
</script>

<style scoped lang="less">
.drafting { width: 100%; height: 100%; overflow: auto; padding: 34px 42px 54px; box-sizing: border-box; color: var(--legal-text-primary); background: var(--legal-bg-page); }
.drafting__header, .drafting__grid, .jobs { max-width: 1320px; margin-left: auto; margin-right: auto; }
.drafting__header { display: flex; justify-content: space-between; align-items: end; margin-bottom: 22px; }
.eyebrow { color: var(--legal-text-secondary); font-size: 10px; letter-spacing: .12em; text-transform: uppercase; }
h1 { margin: 7px 0 5px; font-size: 28px; letter-spacing: -.03em; } h2 { margin: 0 0 16px; font-size: 15px; } p { margin: 0; color: var(--legal-text-secondary); font-size: 13px; }
.drafting__grid { display: grid; grid-template-columns: minmax(0, 1.4fr) minmax(280px, .8fr); gap: 16px; }
.panel { border: 1px solid var(--legal-border); border-radius: 6px; background: var(--legal-bg-surface); padding: 22px; }
.form { display: grid; gap: 14px; } label { display: grid; gap: 6px; font-size: 12px; font-weight: 650; } input, textarea, select { width: 100%; box-sizing: border-box; border: 1px solid var(--legal-border); border-radius: 4px; padding: 9px; color: var(--legal-text-primary); background: var(--legal-bg-page); font: inherit; } textarea { resize: vertical; }
.primary, .secondary { border-radius: 4px; padding: 9px 14px; cursor: pointer; font-weight: 650; } .primary { border: 1px solid var(--legal-brand); color: white; background: var(--legal-brand); } .secondary { border: 1px solid var(--legal-border); color: var(--legal-text-primary); background: transparent; } button:disabled { opacity: .5; cursor: not-allowed; }
.hint { font-size: 11px; line-height: 1.5; } .error { margin-top: 8px; color: var(--legal-risk-strong); font-size: 12px; }
.engine-row { display: flex; align-items: center; justify-content: space-between; padding: 11px 0; border-bottom: 1px solid var(--legal-border); } .engine-row div span { display: block; margin-top: 3px; color: var(--legal-text-secondary); font-size: 10px; } .engine-row b { font-size: 10px; text-transform: uppercase; } .ok { color: var(--legal-ai-strong); } .muted { color: var(--legal-text-secondary); }
.jobs { margin-top: 16px; border: 1px solid var(--legal-border); border-radius: 6px; background: var(--legal-bg-surface); overflow: visible; }
.jobs__title { min-height: 62px; padding: 0 16px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--legal-border); } .jobs__title h2 { display: inline; margin: 0 8px 0 0; } .jobs__title span { color: var(--legal-text-secondary); font-size: 11px; }
.jobs__filters { padding: 12px; display: grid; grid-template-columns: minmax(240px, 1fr) minmax(280px, auto) 150px 175px auto; gap: 8px; border-bottom: 1px solid var(--legal-border); }
.filter, .date-filter, .status-filter summary { min-height: 38px; box-sizing: border-box; border: 1px solid var(--legal-border); border-radius: 5px; background: var(--legal-bg-page); }
.filter { display: flex; align-items: center; gap: 8px; padding: 0 10px; color: var(--legal-text-secondary); } .filter input, .filter select { min-width: 0; padding: 0; border: 0; outline: 0; background: transparent; } .filter--select { padding: 0 8px; }
.date-filter { display: flex; align-items: center; gap: 7px; padding: 0 10px; color: var(--legal-text-secondary); } .date-filter input { width: 116px; padding: 0; border: 0; background: transparent; outline: 0; color: var(--legal-text-primary); font-size: 11px; }
.status-filter { position: relative; } .status-filter summary { height: 38px; padding: 0 10px; display: flex; align-items: center; gap: 8px; list-style: none; cursor: pointer; font-size: 11px; } .status-filter summary::-webkit-details-marker { display: none; } .status-filter summary > :last-child { margin-left: auto; }
.status-dots { display: flex; } .status-dots i { margin-left: -2px; border: 2px solid var(--legal-bg-page); }
.status-filter__menu { position: absolute; z-index: 20; top: 43px; right: 0; width: 190px; padding: 6px; border: 1px solid var(--legal-border); border-radius: 6px; background: var(--legal-bg-surface); box-shadow: 0 10px 30px rgba(0,0,0,.12); } .status-filter__menu label { padding: 7px 8px; display: flex; grid-template-columns: none; align-items: center; gap: 8px; border-radius: 4px; cursor: pointer; font-weight: 500; } .status-filter__menu label:hover { background: var(--legal-bg-hover); } .status-filter__menu input { width: 14px; margin: 0; accent-color: var(--legal-brand); }
.status-filter i, .status-dots i { width: 9px; height: 9px; flex: none; border-radius: 50%; } .dot--queued { background: var(--legal-status-queued); } .dot--running { background: var(--legal-status-running); } .dot--completed { background: var(--legal-status-completed); } .dot--failed { background: var(--legal-status-failed); } .dot--cancelled { background: var(--legal-status-cancelled); }
.clear-filters { align-self: center; border: 0; color: var(--legal-brand); background: transparent; cursor: pointer; font-size: 11px; white-space: nowrap; }
.jobs__table { overflow: hidden; border-radius: 0 0 6px 6px; } .jobs__head, .job { display: grid; grid-template-columns: minmax(300px, 2fr) 150px 110px 110px 150px 90px; align-items: center; }
.jobs__head { height: 36px; padding: 0 14px; color: var(--legal-text-secondary); background: var(--legal-bg-hover); font-size: 10px; text-transform: uppercase; letter-spacing: .04em; }
.job { position: relative; min-height: 66px; padding: 0 14px; border-top: 1px solid var(--legal-border); cursor: pointer; outline: 0; } .job:hover, .job:has(.job__link:focus-visible) { background: var(--legal-bg-hover); } .job__link { position: absolute; z-index: 1; inset: 0; border-radius: 4px; } .job__link:focus-visible { outline: 2px solid var(--legal-ai); outline-offset: -3px; }
.job__main { min-width: 0; display: flex; align-items: center; gap: 10px; } .job__main > i { width: 32px; height: 38px; display: flex; align-items: center; justify-content: center; flex: none; border: 1px solid var(--legal-border); border-radius: 4px; color: var(--legal-ai-strong); background: var(--legal-ai-soft); } .job__main div { min-width: 0; } .job__main strong, .job__instruction, .job__id { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; } .job__main strong { font-size: 12px; } .job__instruction { max-width: 460px; margin-top: 4px; color: var(--legal-text-secondary); font-size: 10px; } .job__id { width: fit-content; max-width: 100%; margin-top: 5px; color: var(--legal-text-secondary); font: 10px/1.2 ui-monospace, SFMono-Regular, Menlo, monospace; }
.job__status { display: flex; align-items: center; gap: 6px; min-width: 0; } .job__status b { font-size: 10px; font-weight: 700; white-space: nowrap; } .job__status small { color: var(--legal-text-secondary); font-size: 10px; white-space: nowrap; } .job__status--queued b { color: var(--legal-status-queued-strong); } .job__status--running b { color: var(--legal-status-running-strong); } .job__status--completed b { color: var(--legal-status-completed-strong); } .job__status--failed b { color: var(--legal-status-failed-strong); } .job__status--cancelled b { color: var(--legal-status-cancelled-strong); }
.status-indicator { width: 9px; height: 9px; flex: none; border-radius: 50%; background: var(--legal-status-cancelled); } .status-indicator--queued { background: var(--legal-status-queued); box-shadow: 0 0 0 3px var(--legal-status-queued-soft); } .status-indicator--running { background: var(--legal-status-running); box-shadow: 0 0 0 3px var(--legal-status-running-soft); animation: drafting-status-pulse 1.8s ease-out infinite; } .status-indicator--completed { background: var(--legal-status-completed); box-shadow: 0 0 0 3px var(--legal-status-completed-soft); } .status-indicator--failed { background: var(--legal-status-failed); box-shadow: 0 0 0 3px var(--legal-status-failed-soft); } .status-indicator--cancelled { box-shadow: 0 0 0 3px var(--legal-status-cancelled-soft); }
@keyframes drafting-status-pulse { 0%, 100% { box-shadow: 0 0 0 3px var(--legal-status-running-soft); } 50% { box-shadow: 0 0 0 6px transparent; } }
.mode-pill { justify-self: start; padding: 4px 7px; border-radius: 10px; color: var(--legal-brand); background: var(--legal-ai-soft); font-size: 10px; } .job__artifacts, .job time { color: var(--legal-text-secondary); font-size: 10px; }
.job__actions { position: relative; z-index: 2; display: flex; justify-content: flex-end; gap: 2px; } .job__actions button { width: 27px; height: 27px; border: 0; border-radius: 4px; color: var(--legal-text-secondary); background: transparent; cursor: pointer; } .job__actions button:hover { color: var(--legal-text-primary); background: var(--legal-bg-page); }
.jobs__empty { min-height: 220px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; color: var(--legal-text-secondary); font-size: 12px; } .jobs__empty strong { color: var(--legal-text-primary); font-size: 13px; } .jobs__empty--error { color: var(--legal-risk-strong); }
@media (max-width: 1100px) { .jobs__filters { grid-template-columns: 1fr 1fr; } .jobs__head, .job { grid-template-columns: minmax(260px, 2fr) 140px 100px 120px 80px; } .jobs__head > :nth-child(4), .job > :nth-child(4) { display: none; } }
@media (max-width: 700px) { .drafting { padding: 24px 18px 40px; } .drafting__grid { grid-template-columns: 1fr; } .jobs__filters { grid-template-columns: 1fr; } .date-filter input { width: 100%; } .jobs__head { display: none; } .job { position: relative; grid-template-columns: 1fr auto; gap: 8px; padding: 13px; } .job__main { grid-column: 1 / -1; } .job__status { grid-column: 1; } .mode-pill { grid-column: 2; grid-row: 2; } .job__artifacts, .job time { display: none; } .job__actions { position: absolute; right: 8px; top: 10px; padding-left: 20px; background: linear-gradient(90deg, transparent, var(--legal-bg-surface) 20%); } .job:hover .job__actions { background: linear-gradient(90deg, transparent, var(--legal-bg-hover) 20%); } }
</style>
