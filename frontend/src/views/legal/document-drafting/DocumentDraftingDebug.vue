<template>
  <section class="debug-page">
    <header class="debug-topbar">
      <button type="button" class="link-button" @click="backToDetail"><t-icon name="chevron-left" /> {{ t('documentDrafting.debug.back') }}</button>
      <div><strong>{{ t('documentDrafting.debug.title') }}</strong><code v-if="job">#{{ shortDocumentEditId(job.id) }}</code></div>
      <button type="button" class="primary" :disabled="!job || !isTerminal(job.status)" @click="openCompare"><t-icon name="git-compare" /> {{ t('documentDrafting.debug.compare') }}</button>
    </header>

    <main class="debug-content">
      <div v-if="loading && !debug" class="page-state"><t-loading /> {{ t('documentDrafting.debug.loading') }}</div>
      <div v-else-if="error || !debug || !job" class="page-state page-state--error"><t-icon name="error-circle" /><strong>{{ t('documentDrafting.debug.loadFailed') }}</strong><p>{{ error }}</p><button class="secondary" @click="() => load()">{{ t('documentDrafting.retry') }}</button></div>
      <template v-else>
        <section class="debug-heading">
          <div><span class="eyebrow">{{ t('documentDrafting.debug.eyebrow') }}</span><h1>{{ job.file_name }}</h1><p>{{ job.instruction }}</p></div>
          <span class="status" :class="`status--${job.status}`">{{ t(`documentDrafting.status.${job.status}`) }}</span>
        </section>

        <section class="summary-grid">
          <article><span>{{ t('documentDrafting.debug.mode') }}</span><strong>{{ modeLabel(job.mode) }}</strong><small>{{ job.comparison_strategy || '—' }}</small></article>
          <article><span>Worker / Protocol</span><strong>{{ workerSummary }}</strong><small>{{ protocolSummary }}</small></article>
          <article><span>{{ t('documentDrafting.debug.model') }}</span><strong>{{ debug.model?.display_name || debug.model?.name || job.model_id || '—' }}</strong><small>{{ debug.model?.id || '—' }}</small></article>
          <article><span>{{ t('documentDrafting.debug.duration') }}</span><strong>{{ formatDuration(totalDuration) }}</strong><small>{{ debug.stages.length }} {{ t('documentDrafting.debug.stageUnit') }}</small></article>
          <article><span>SHA-256</span><strong class="mono">{{ job.source_sha256.slice(0, 16) }}…</strong><small>{{ formatBytes(job.file_size) }}</small></article>
        </section>

        <div v-if="!debug.trace_recorded" class="legacy-notice"><t-icon name="info-circle" />{{ t('documentDrafting.debug.notRecorded') }}</div>

        <section class="panel">
          <header><div><span>01</span><h2>{{ t('documentDrafting.debug.timeline') }}</h2></div><small>{{ active ? t('documentDrafting.debug.autoRefreshing') : t('documentDrafting.debug.terminal') }}</small></header>
          <div v-if="debug.stages.length" class="stage-list">
            <article v-for="stage in debug.stages" :key="stage.id" :class="`stage stage--${stage.status}`">
              <i /><div class="stage-main"><strong>{{ stage.stage }}</strong><span>{{ stage.engine_name || 'LexAI' }}<template v-if="stage.engine_version"> · {{ stage.engine_version }}</template></span><p v-if="stage.error_message">{{ stage.error_code }} · {{ stage.error_message }}</p></div>
              <div class="stage-meta"><span>{{ t('documentDrafting.debug.attempt', { count: stage.attempt }) }}</span><strong>{{ formatDuration(stage.duration_ms) }}</strong></div>
            </article>
          </div>
          <p v-else class="empty">{{ t('documentDrafting.debug.notRecorded') }}</p>
        </section>

        <section class="panel split-panel">
          <header><div><span>02</span><h2>Inspect</h2></div><small>{{ inspectSummary }}</small></header>
          <div class="split-content">
            <div class="facts"><dl><div><dt>{{ t('documentDrafting.debug.sourceEngine') }}</dt><dd>{{ latestInspect?.engine_name || '—' }}</dd></div><div><dt>{{ t('documentDrafting.debug.characters') }}</dt><dd>{{ latestInspect?.output_summary?.characters ?? '—' }}</dd></div><div><dt>{{ t('documentDrafting.debug.truncated') }}</dt><dd>{{ planInput?.truncated ? t('documentDrafting.debug.yes') : t('documentDrafting.debug.no') }}</dd></div></dl></div>
            <div class="blob-area"><button v-for="blob in blobsByKind('inspect_text')" :key="blob.id" class="secondary" @click="loadBlob(blob)">{{ t('documentDrafting.debug.loadSnapshot') }} · {{ formatBytes(blob.size) }}</button><pre v-if="firstBlobContent('inspect_text')">{{ firstBlobContent('inspect_text') }}</pre><p v-else class="empty">{{ t('documentDrafting.debug.expandToLoad') }}</p></div>
          </div>
        </section>

        <section class="panel">
          <header><div><span>03</span><h2>Planner</h2></div><small>{{ planStage?.input_summary?.prompt_version || '—' }}</small></header>
          <div class="parameter-grid"><div><span>Model</span><strong>{{ job.model_id || '—' }}</strong></div><div><span>Temperature</span><strong>{{ planInput?.temperature ?? '—' }}</strong></div><div><span>Max tokens</span><strong>{{ planInput?.max_completion_tokens ?? '—' }}</strong></div><div><span>Finish reason</span><strong>{{ planOutput?.finish_reason || '—' }}</strong></div><div><span>Repairs</span><strong>{{ planOutput?.repair_count ?? '—' }}</strong></div><div><span>Token usage</span><strong>{{ tokenTotal(planOutput?.usage) }}</strong></div></div>
          <div class="blob-buttons"><button v-for="blob in plannerBlobs" :key="blob.id" class="secondary" @click="loadBlob(blob)">{{ blob.kind }} · {{ formatBytes(blob.size) }}</button></div>
          <details v-for="blob in plannerBlobs.filter(item => blobContents[item.id])" :key="`loaded-${blob.id}`"><summary>{{ blob.kind }}</summary><pre>{{ prettyBlob(blob) }}</pre></details>
        </section>

        <section class="panel">
          <header><div><span>04</span><h2>EditPlan</h2></div><small>{{ planOperations.length }} {{ t('documentDrafting.debug.operationUnit') }}</small></header>
          <div v-if="planOperations.length" class="table-wrap"><table><thead><tr><th>ID</th><th>{{ t('documentDrafting.debug.kind') }}</th><th>Target</th><th>Payload</th><th>Expected</th><th>Actual</th><th>Engine</th><th>{{ t('documentDrafting.debug.result') }}</th></tr></thead><tbody><tr v-for="operation in planOperations" :key="operation.operation_id"><td><code>{{ operation.operation_id }}</code></td><td>{{ operation.kind }}</td><td class="quote">{{ operation.target?.quote }}</td><td class="quote">{{ operation.payload?.text || operation.payload?.comment || '—' }}</td><td>{{ operation.target?.expected_matches }}</td><td>{{ operationRecord(operation.operation_id)?.actual_matches ?? '—' }}</td><td>{{ operationRecord(operation.operation_id)?.engine_name || '—' }}</td><td><span :class="`op-result op-result--${operationRecord(operation.operation_id)?.status || 'planned'}`">{{ operationRecord(operation.operation_id)?.status || 'planned' }}</span><small v-if="operationRecord(operation.operation_id)?.engine_message">{{ operationRecord(operation.operation_id)?.engine_message }}</small></td></tr></tbody></table></div>
          <p v-else class="empty">{{ t('documentDrafting.noOperations') }}</p>
          <details><summary>{{ t('documentDrafting.debug.planJson') }}</summary><pre>{{ JSON.stringify(job.plan || {}, null, 2) }}</pre></details>
        </section>

        <section class="panel">
          <header><div><span>05</span><h2>{{ t('documentDrafting.debug.resultAnalysis') }}</h2></div><small>{{ job.artifacts?.length || 0 }} {{ t('documentDrafting.debug.artifactUnit') }}</small></header>
          <div class="result-grid"><article><span>{{ t('documentDrafting.debug.artifacts') }}</span><ul><li v-for="artifact in job.artifacts || []" :key="artifact.id"><strong>{{ artifact.kind }}</strong><code>{{ artifact.sha256.slice(0, 12) }}…</code><small>{{ formatBytes(artifact.size) }}</small></li></ul></article><article><span>{{ t('documentDrafting.debug.validation') }}</span><pre>{{ validationSummary }}</pre></article></div>
          <button v-if="sourceTextBlob && cleanTextBlob" class="secondary" @click="loadDiff">{{ diffLoading ? t('documentDrafting.debug.loadingDiff') : t('documentDrafting.debug.loadDiff') }}</button>
          <pre v-if="textDiff" class="diff">{{ textDiff }}</pre><p v-else-if="!cleanTextBlob" class="empty">{{ t('documentDrafting.debug.diffNotRecorded') }}</p>
        </section>

        <section class="panel">
          <header><div><span>06</span><h2>{{ t('documentDrafting.debug.comparison') }}</h2></div><button class="secondary" :disabled="!isTerminal(job.status)" @click="openCompare">{{ t('documentDrafting.debug.newComparison') }}</button></header>
          <div v-if="comparison?.jobs?.length" class="comparison-grid"><article v-for="item in comparison.jobs" :key="item.id" :class="{ current: item.id === job.id }"><header><strong>{{ modeLabel(item.mode) }}</strong><span :class="`status status--${item.status}`">{{ t(`documentDrafting.status.${item.status}`) }}</span></header><dl><div><dt>{{ t('documentDrafting.debug.strategy') }}</dt><dd>{{ item.comparison_strategy || t('documentDrafting.debug.sourceTask') }}</dd></div><div><dt>{{ t('documentDrafting.debug.duration') }}</dt><dd>{{ formatDuration(jobDuration(item)) }}</dd></div><div><dt>Plan</dt><dd>{{ planOperationCount(item) }} ops</dd></div><div><dt>Matches</dt><dd>{{ matchSummary(item) }}</dd></div><div><dt>Tokens</dt><dd>{{ comparisonTokens(item.id) }}</dd></div><div><dt>Artifacts</dt><dd>{{ item.artifacts?.length || 0 }}</dd></div><div><dt>Issues</dt><dd>{{ comparisonIssues(item) }}</dd></div></dl><button v-if="item.id !== job.id" class="link-button" @click="openJobDebug(item.id)">{{ t('documentDrafting.debug.openTask') }}</button></article></div>
          <p v-else class="empty">{{ t('documentDrafting.debug.noComparison') }}</p>
        </section>
      </template>
    </main>

    <div v-if="compareOpen" class="modal-backdrop" @click.self="compareOpen = false"><section class="compare-modal"><header><div><h2>{{ t('documentDrafting.debug.compareTitle') }}</h2><p>{{ t('documentDrafting.debug.compareHint') }}</p></div><button class="icon-button" @click="compareOpen = false"><t-icon name="close" /></button></header><div class="modal-section"><strong>{{ t('documentDrafting.debug.strategy') }}</strong><label><input v-model="compareStrategy" type="radio" value="replan"> <span><b>replan</b><small>{{ t('documentDrafting.debug.replanHint') }}</small></span></label><label><input v-model="compareStrategy" type="radio" value="locked_plan"> <span><b>locked_plan</b><small>{{ t('documentDrafting.debug.lockedHint') }}</small></span></label></div><div class="modal-section"><strong>{{ t('documentDrafting.debug.modes') }}</strong><label v-for="mode in allModes" :key="mode"><input v-model="compareModes" type="checkbox" :value="mode"> <span><b>{{ modeLabel(mode) }}</b></span></label></div><footer><button class="secondary" @click="compareOpen = false">{{ t('documentDrafting.cancel') }}</button><button class="primary" :disabled="comparing || !compareModes.length" @click="startComparison">{{ comparing ? t('documentDrafting.debug.startingComparison') : t('documentDrafting.debug.startComparison') }}</button></footer></section></div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'

import { createDocumentEditComparison, getDocumentEditComparison, getDocumentEditDebug, getDocumentEditDebugBlob, type DocumentEditComparison, type DocumentEditComparisonStrategy, type DocumentEditDebug, type DocumentEditDebugBlob, type DocumentEditJob, type DocumentEditMode } from '@/api/document-edit'
import { LEGAL_DOCUMENT_DRAFTING_DEBUG_ROUTE, LEGAL_DOCUMENT_DRAFTING_DETAIL_ROUTE } from '@/router/paths'
import { shortDocumentEditId } from '../documentDraftingTasks'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const debug = ref<DocumentEditDebug | null>(null)
const comparison = ref<DocumentEditComparison | null>(null)
const loading = ref(true)
const error = ref('')
const compareOpen = ref(false)
const comparing = ref(false)
const compareStrategy = ref<DocumentEditComparisonStrategy>('replan')
const compareModes = ref<DocumentEditMode[]>([])
const blobContents = reactive<Record<string, string>>({})
const comparisonDebug = reactive<Record<string, DocumentEditDebug>>({})
const diffLoading = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

const allModes: DocumentEditMode[] = ['adeu', 'officecli', 'hybrid']
const job = computed(() => debug.value?.job || null)
const active = computed(() => Boolean(job.value && !isTerminal(job.value.status)) || Boolean(comparison.value?.jobs?.some(item => !isTerminal(item.status))))
const totalDuration = computed(() => debug.value?.stages.reduce((sum, stage) => sum + (stage.duration_ms || 0), 0) || jobDuration(job.value))
const planStage = computed(() => [...(debug.value?.stages || [])].reverse().find(stage => stage.stage === 'Plan'))
const latestInspect = computed(() => [...(debug.value?.stages || [])].reverse().find(stage => stage.stage === 'Inspect' && stage.status === 'completed'))
const planInput = computed(() => planStage.value?.input_summary || {})
const planOutput = computed(() => planStage.value?.output_summary || {})
const plannerBlobs = computed(() => (debug.value?.blobs || []).filter(blob => blob.kind.startsWith('planner_')))
const planOperations = computed<any[]>(() => Array.isArray((job.value?.plan as any)?.operations) ? (job.value?.plan as any).operations : [])
const sourceTextBlob = computed(() => (debug.value?.blobs || []).find(blob => blob.kind === 'inspect_text'))
const cleanTextBlob = computed(() => (debug.value?.blobs || []).find(blob => blob.kind === 'clean_text'))
const inspectSummary = computed(() => latestInspect.value ? `${latestInspect.value.engine_name || '—'} · ${latestInspect.value.duration_ms}ms` : t('documentDrafting.debug.notRecorded'))
const validationSummary = computed(() => JSON.stringify([...((debug.value?.stages || []).filter(stage => stage.stage === 'Validate'))].pop()?.output_summary || {}, null, 2))
const textDiff = computed(() => sourceTextBlob.value && cleanTextBlob.value && blobContents[sourceTextBlob.value.id] && blobContents[cleanTextBlob.value.id] ? lineDiff(blobContents[sourceTextBlob.value.id], blobContents[cleanTextBlob.value.id]) : '')
const workerSummary = computed(() => Object.values(job.value?.capabilities || {}).map(value => `${value.engine_name || 'worker'} ${value.engine_version || ''}`.trim()).join(' + ') || '—')
const protocolSummary = computed(() => [...new Set(Object.values(job.value?.capabilities || {}).map(value => value.protocol_version).filter(Boolean))].join(', ') || '—')

function isTerminal(status: string) { return ['completed', 'failed', 'cancelled'].includes(status) }
function modeLabel(mode: DocumentEditMode) { return mode === 'adeu' ? 'Adeu' : mode === 'officecli' ? 'OfficeCLI' : 'Hybrid' }
function formatDuration(ms: number) { if (!ms && ms !== 0) return '—'; if (ms < 1000) return `${ms} ms`; if (ms < 60000) return `${(ms / 1000).toFixed(1)} s`; return `${Math.floor(ms / 60000)}m ${Math.round((ms % 60000) / 1000)}s` }
function formatBytes(value = 0) { if (value < 1024) return `${value} B`; if (value < 1048576) return `${(value / 1024).toFixed(1)} KB`; return `${(value / 1048576).toFixed(1)} MB` }
function jobDuration(value: DocumentEditJob | null) { if (!value) return 0; return Math.max(0, new Date(value.completed_at || value.updated_at).getTime() - new Date(value.started_at || value.created_at).getTime()) }
function tokenTotal(usage: any) { if (!usage) return '—'; if (usage.total_tokens !== undefined) return usage.total_tokens; if (usage.TotalTokens !== undefined) return usage.TotalTokens; let total = 0; Object.values(usage).forEach((value) => { if (typeof value === 'number') total += value }); return total || '—' }
function blobsByKind(kind: string) { return (debug.value?.blobs || []).filter(blob => blob.kind === kind) }
function firstBlobContent(kind: string) { const blob = blobsByKind(kind)[0]; return blob ? blobContents[blob.id] : '' }
function prettyBlob(blob: DocumentEditDebugBlob) { const value = blobContents[blob.id] || ''; try { return JSON.stringify(JSON.parse(value), null, 2) } catch { return value } }
function operationRecord(id: string) { return job.value?.operations?.find(item => item.operation_id === id) }
function matchSummary(value: DocumentEditJob) { const rows = value.operations || []; const known = rows.filter(row => row.actual_matches !== undefined); return known.length ? `${known.filter(row => row.actual_matches === row.expected_matches).length}/${known.length}` : '—' }
function planOperationCount(value: DocumentEditJob) { const operations = (value.plan as any)?.operations; return Array.isArray(operations) ? operations.length : 0 }
function comparisonTokens(id: string) { const stage = [...(comparisonDebug[id]?.stages || [])].reverse().find(item => item.stage === 'Plan'); return tokenTotal(stage?.output_summary?.usage) }
function comparisonIssues(value: DocumentEditJob) { const validate = [...(comparisonDebug[value.id]?.stages || [])].reverse().find(item => item.stage === 'Validate'); return value.error_code || validate?.error_code || (validate?.output_summary?.warnings ? `${validate.output_summary.warnings} warning(s)` : '—') }
function lineDiff(before: string, after: string) { const left = before.split('\n'); const right = after.split('\n'); const result: string[] = []; const max = Math.max(left.length, right.length); for (let index = 0; index < max; index += 1) { if (left[index] === right[index]) { if (left[index] !== undefined) result.push(`  ${left[index]}`) } else { if (left[index] !== undefined) result.push(`- ${left[index]}`); if (right[index] !== undefined) result.push(`+ ${right[index]}`) } } return result.join('\n') }

async function loadBlob(blob: DocumentEditDebugBlob) { if (blobContents[blob.id]) return; try { blobContents[blob.id] = await getDocumentEditDebugBlob(blob.job_id, blob.stage_run_id, blob.kind) } catch (cause: any) { MessagePlugin.error(cause?.message || t('documentDrafting.debug.blobLoadFailed')) } }
async function loadDiff() { if (!sourceTextBlob.value || !cleanTextBlob.value) return; diffLoading.value = true; await Promise.all([loadBlob(sourceTextBlob.value), loadBlob(cleanTextBlob.value)]); diffLoading.value = false }
async function load(silent = false) { const id = String(route.params.jobId || ''); if (!silent) loading.value = true; error.value = ''; try { const [debugResponse, comparisonResponse] = await Promise.all([getDocumentEditDebug(id), getDocumentEditComparison(id)]); debug.value = debugResponse.data; comparison.value = comparisonResponse.data; await Promise.all((comparison.value.jobs || []).map(async (item) => { try { comparisonDebug[item.id] = (await getDocumentEditDebug(item.id)).data } catch { /* owner-filtered task may disappear while refreshing */ } })) } catch (cause: any) { error.value = cause?.message || t('documentDrafting.debug.loadFailed') } finally { if (!silent) loading.value = false; schedulePoll() } }
function schedulePoll() { if (pollTimer) clearInterval(pollTimer); pollTimer = null; if (active.value) pollTimer = setInterval(() => { void load(true) }, 2000) }
function backToDetail() { void router.push({ name: LEGAL_DOCUMENT_DRAFTING_DETAIL_ROUTE, params: { jobId: route.params.jobId } }) }
function openJobDebug(id: string) { void router.push({ name: LEGAL_DOCUMENT_DRAFTING_DEBUG_ROUTE, params: { jobId: id } }) }
function openCompare() { if (!job.value || !isTerminal(job.value.status)) return; const covered = new Set((comparison.value?.jobs || [job.value]).map(item => item.mode)); compareModes.value = allModes.filter(mode => !covered.has(mode)); if (!compareModes.value.length) compareModes.value = allModes.filter(mode => mode !== job.value?.mode); if (!compareModes.value.length) compareModes.value = [...allModes]; compareStrategy.value = 'replan'; compareOpen.value = true }
async function startComparison() { if (!job.value || !compareModes.value.length) return; comparing.value = true; try { comparison.value = (await createDocumentEditComparison(job.value.id, compareModes.value, compareStrategy.value)).data; compareOpen.value = false; schedulePoll() } catch (cause: any) { MessagePlugin.error(cause?.message || t('documentDrafting.debug.compareFailed')) } finally { comparing.value = false } }

watch(() => route.params.jobId, () => { Object.keys(blobContents).forEach(key => delete blobContents[key]); void load() })
onMounted(() => { void load() })
onBeforeUnmount(() => { if (pollTimer) clearInterval(pollTimer) })
</script>

<style scoped lang="less">
.debug-page { width: 100%; height: 100%; overflow: auto; color: var(--legal-text-primary); background: var(--legal-bg-page); }
.debug-topbar { position: sticky; z-index: 6; top: 0; min-height: 54px; padding: 0 24px; display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; border-bottom: 1px solid var(--legal-border); background: var(--legal-bg-surface); } .debug-topbar > div { display: flex; align-items: center; gap: 9px; font-size: 12px; } .debug-topbar code { color: var(--legal-text-secondary); font-size: 10px; } .debug-topbar > :last-child { justify-self: end; }
.debug-content { width: min(1380px, calc(100% - 48px)); margin: 0 auto; padding: 30px 0 60px; }
.debug-heading { display: flex; justify-content: space-between; gap: 24px; align-items: flex-start; margin-bottom: 22px; } .eyebrow { color: var(--legal-ai-strong); font-size: 9px; font-weight: 800; letter-spacing: .14em; text-transform: uppercase; } .debug-heading h1 { margin: 6px 0; font-size: 24px; } .debug-heading p { max-width: 880px; margin: 0; color: var(--legal-text-secondary); font-size: 11px; line-height: 1.6; }
.primary,.secondary,.link-button,.icon-button { min-height: 32px; display: inline-flex; align-items: center; justify-content: center; gap: 5px; border-radius: 5px; cursor: pointer; font-size: 10px; } .primary { padding: 0 12px; border: 1px solid var(--legal-brand); color: #fff; background: var(--legal-brand); } .secondary { padding: 0 11px; border: 1px solid var(--legal-border); color: var(--legal-text-primary); background: var(--legal-bg-surface); } .link-button,.icon-button { padding: 0; border: 0; color: var(--legal-brand); background: transparent; } button:disabled { opacity: .45; cursor: not-allowed; }
.status { padding: 5px 9px; border-radius: 12px; font-size: 9px; font-weight: 750; background: var(--legal-status-cancelled-soft); } .status--running,.status--queued { color: var(--legal-status-running-strong); background: var(--legal-status-running-soft); } .status--completed { color: var(--legal-status-completed-strong); background: var(--legal-status-completed-soft); } .status--failed { color: var(--legal-status-failed-strong); background: var(--legal-status-failed-soft); }
.summary-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 10px; margin-bottom: 14px; } .summary-grid article { min-width: 0; padding: 16px; border: 1px solid var(--legal-border); border-radius: 6px; background: var(--legal-bg-surface); } .summary-grid span,.summary-grid small { display: block; color: var(--legal-text-secondary); font-size: 9px; } .summary-grid strong { display: block; margin: 8px 0 5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 13px; } .mono { font-family: ui-monospace, monospace; }
.legacy-notice { margin-bottom: 14px; padding: 11px 14px; display: flex; align-items: center; gap: 8px; border: 1px solid var(--legal-border); border-radius: 6px; color: var(--legal-text-secondary); background: var(--legal-ai-soft); font-size: 10px; }
.panel { margin-top: 12px; border: 1px solid var(--legal-border); border-radius: 7px; background: var(--legal-bg-surface); overflow: hidden; } .panel > header { min-height: 52px; padding: 0 16px; display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid var(--legal-border); } .panel > header > div { display: flex; align-items: center; gap: 9px; } .panel > header span { color: var(--legal-ai-strong); font: 9px ui-monospace, monospace; } .panel h2 { margin: 0; font-size: 12px; } .panel > header small { color: var(--legal-text-secondary); font-size: 9px; } .panel > pre,.panel > details,.panel > .blob-buttons,.panel > .parameter-grid,.panel > .table-wrap,.panel > .result-grid,.panel > button,.panel > .diff,.panel > .empty { margin: 16px; }
.stage-list { padding: 4px 18px 12px; } .stage { position: relative; min-height: 54px; display: grid; grid-template-columns: 14px 1fr auto; gap: 8px; align-items: center; } .stage:not(:last-child)::after { content:''; position:absolute; left:4px; top:34px; bottom:-20px; width:1px; background:var(--legal-border); } .stage > i { z-index:1; width:9px; height:9px; border-radius:50%; background:var(--legal-status-running); box-shadow:0 0 0 4px var(--legal-bg-surface); } .stage--completed > i { background:var(--legal-status-completed); } .stage--failed > i { background:var(--legal-status-failed); } .stage--skipped > i { background:var(--legal-status-cancelled); } .stage-main strong,.stage-main span { display:block; font-size:10px; } .stage-main span { margin-top:3px; color:var(--legal-text-secondary); } .stage-main p { margin:4px 0 0; color:var(--legal-status-failed-strong); font-size:9px; } .stage-meta { text-align:right; } .stage-meta span,.stage-meta strong { display:block; font-size:9px; } .stage-meta span { color:var(--legal-text-secondary); }
.split-content { display:grid; grid-template-columns:280px 1fr; } .facts { padding:16px; border-right:1px solid var(--legal-border); } dl { margin:0; } dl div { margin-bottom:12px; } dt { color:var(--legal-text-secondary); font-size:9px; } dd { margin:4px 0 0; font-size:10px; overflow-wrap:anywhere; } .blob-area { padding:16px; } pre { max-height:420px; padding:14px; overflow:auto; border:1px solid var(--legal-border); border-radius:5px; color:var(--legal-text-primary); background:var(--legal-bg-paper); white-space:pre-wrap; overflow-wrap:anywhere; font:10px/1.65 ui-monospace, SFMono-Regular, Menlo, monospace; }
.parameter-grid { display:grid; grid-template-columns:repeat(6,1fr); gap:8px; } .parameter-grid div { padding:10px; border:1px solid var(--legal-border); border-radius:5px; } .parameter-grid span,.parameter-grid strong { display:block; font-size:9px; } .parameter-grid span { color:var(--legal-text-secondary); } .parameter-grid strong { margin-top:5px; } .blob-buttons { display:flex; flex-wrap:wrap; gap:7px; } details { border-top:1px solid var(--legal-border); } details summary { padding:11px 0; cursor:pointer; font-size:10px; }
.table-wrap { overflow:auto; } table { width:100%; border-collapse:collapse; font-size:9px; } th,td { padding:9px 8px; border-bottom:1px solid var(--legal-border); text-align:left; vertical-align:top; } th { color:var(--legal-text-secondary); font-weight:650; } td.quote { min-width:180px; max-width:320px; white-space:pre-wrap; } td small { display:block; max-width:180px; margin-top:4px; color:var(--legal-text-secondary); } .op-result { font-weight:700; } .op-result--applied { color:var(--legal-status-completed-strong); } .op-result--failed { color:var(--legal-status-failed-strong); }
.result-grid { display:grid; grid-template-columns:1fr 1fr; gap:12px; } .result-grid article { min-width:0; padding:14px; border:1px solid var(--legal-border); border-radius:5px; } .result-grid > article > span { color:var(--legal-text-secondary); font-size:9px; } .result-grid ul { margin:10px 0 0; padding:0; list-style:none; } .result-grid li { padding:7px 0; display:grid; grid-template-columns:1fr auto auto; gap:10px; border-top:1px solid var(--legal-border); font-size:9px; } .diff { white-space:pre; }
.comparison-grid { padding:14px; display:grid; grid-template-columns:repeat(3,1fr); gap:10px; } .comparison-grid article { padding:13px; border:1px solid var(--legal-border); border-radius:5px; } .comparison-grid article.current { border-color:var(--legal-brand); } .comparison-grid article > header { display:flex; align-items:center; justify-content:space-between; } .comparison-grid dl { margin-top:14px; display:grid; grid-template-columns:1fr 1fr; gap:8px; } .comparison-grid .link-button { margin-top:8px; }
.empty { color:var(--legal-text-secondary); font-size:10px; } .page-state { min-height:55vh; display:flex; flex-direction:column; align-items:center; justify-content:center; gap:9px; color:var(--legal-text-secondary); font-size:11px; } .page-state--error > :first-child { color:var(--legal-status-failed); }
.modal-backdrop { position:fixed; z-index:20; inset:0; display:flex; align-items:center; justify-content:center; background:rgba(0,0,0,.38); } .compare-modal { width:min(520px,calc(100% - 32px)); border:1px solid var(--legal-border); border-radius:8px; background:var(--legal-bg-surface); box-shadow:0 20px 60px rgba(0,0,0,.2); } .compare-modal > header { padding:18px; display:flex; justify-content:space-between; border-bottom:1px solid var(--legal-border); } .compare-modal h2 { margin:0; font-size:15px; } .compare-modal p { margin:5px 0 0; color:var(--legal-text-secondary); font-size:10px; } .modal-section { padding:15px 18px; display:grid; gap:10px; border-bottom:1px solid var(--legal-border); } .modal-section > strong { font-size:10px; } .modal-section label { display:flex; gap:8px; align-items:flex-start; font-size:10px; } .modal-section label b,.modal-section label small { display:block; } .modal-section label small { margin-top:3px; color:var(--legal-text-secondary); } .compare-modal footer { padding:14px 18px; display:flex; justify-content:flex-end; gap:8px; }
@media(max-width:1000px){.summary-grid{grid-template-columns:1fr 1fr}.parameter-grid{grid-template-columns:repeat(3,1fr)}.comparison-grid{grid-template-columns:1fr}.split-content,.result-grid{grid-template-columns:1fr}.facts{border-right:0;border-bottom:1px solid var(--legal-border)}}
@media(max-width:620px){.debug-topbar{padding:0 12px;grid-template-columns:auto 1fr auto}.debug-topbar code{display:none}.debug-content{width:calc(100% - 24px);padding-top:18px}.summary-grid{grid-template-columns:1fr}.parameter-grid{grid-template-columns:1fr 1fr}.debug-heading{flex-direction:column}.table-wrap{margin-left:8px!important;margin-right:8px!important}}
</style>
