<template>
  <section class="document-viewer">
    <header class="document-viewer__toolbar">
      <div class="document-viewer__file"><t-icon name="file-paste" /> <span>{{ fileName }}</span></div>
      <div class="document-viewer__controls">
        <button type="button" :aria-label="t('contractReview.previousPage')" @click="goToPage(currentPage - 1)"><t-icon name="chevron-up" /></button>
        <span>{{ currentPage }} / {{ pageCount || 1 }}</span>
        <button type="button" :aria-label="t('contractReview.nextPage')" @click="goToPage(currentPage + 1)"><t-icon name="chevron-down" /></button>
        <i />
        <button type="button" :aria-label="t('contractReview.zoomOut')" @click="setZoom(zoom - 0.1)"><t-icon name="zoom-out" /></button>
        <span>{{ Math.round(zoom * 100) }}%</span>
        <button type="button" :aria-label="t('contractReview.zoomIn')" @click="setZoom(zoom + 0.1)"><t-icon name="zoom-in" /></button>
      </div>
    </header>
    <div ref="scrollEl" class="document-viewer__scroll" @scroll="updateCurrentPage">
      <div v-if="loading" class="document-viewer__state"><t-loading size="small" /> {{ t('contractReview.loadingDocument') }}</div>
      <div v-else-if="error" class="document-viewer__state document-viewer__state--error">{{ error }}</div>
      <div v-show="!loading && fileType === '.pdf'" ref="pdfEl" class="document-viewer__pdf" :style="{ '--document-font-compensation': documentFontCompensation }" @click="onDocumentClick" />
      <div v-show="!loading && fileType === '.docx'" ref="docxEl" class="document-viewer__docx" :style="{ '--document-zoom': zoom, '--document-font-compensation': documentFontCompensation }" @click="onDocumentClick" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon as TIcon, Loading as TLoading } from 'tdesign-vue-next'
import { GlobalWorkerOptions, Util, getDocument, type PDFDocumentProxy } from 'pdfjs-dist'
import pdfWorker from 'pdfjs-dist/build/pdf.worker.min.mjs?url'

import { getContractReviewDocument, type ReviewIssue } from '@/api/contract-review'
import { hasReviewQuoteMatch, normalizeReviewText } from './documentLinking'

GlobalWorkerOptions.workerSrc = pdfWorker

const props = defineProps<{ reviewId: string; fileName: string; fileType: string; issues: ReviewIssue[]; selectedIssueId?: string }>()
const emit = defineEmits<{ markerClick: [issueId: string]; locateFailed: [] }>()
const { t } = useI18n()
const scrollEl = ref<HTMLElement | null>(null)
const pdfEl = ref<HTMLElement | null>(null)
const docxEl = ref<HTMLElement | null>(null)
const loading = ref(true)
const error = ref('')
const zoom = ref(1)
const pageCount = ref(0)
const currentPage = ref(1)
const documentFontCompensation = ref(1)
let pdf: PDFDocumentProxy | null = null
let sourceData: ArrayBuffer | null = null
let fontScaleObserver: MutationObserver | null = null

const normalize = normalizeReviewText

function appFontScale() {
  if (typeof window === 'undefined') return 1
  const value = Number.parseFloat(window.getComputedStyle(document.documentElement).zoom)
  return Number.isFinite(value) && value > 0 ? value : 1
}

function updateDocumentFontCompensation() {
  documentFontCompensation.value = Number((1 / appFontScale()).toFixed(4))
}

async function load() {
  loading.value = true; error.value = ''; pdf = null
  try {
    sourceData = await getContractReviewDocument(props.reviewId)
    if (props.fileType === '.pdf') await loadPdf()
    else await loadDocx()
  } catch (cause: any) { error.value = cause?.message || t('contractReview.documentLoadFailed') }
  finally { loading.value = false; await nextTick(); applyIssueMarks() }
}

async function loadPdf() {
  if (!sourceData || !pdfEl.value) return
  pdf = await getDocument({ data: new Uint8Array(sourceData.slice(0)) }).promise
  pageCount.value = pdf.numPages
  await renderPdf()
}

async function renderPdf() {
  if (!pdf || !pdfEl.value) return
  pdfEl.value.innerHTML = ''
  // The document container counteracts the application's CSS zoom, so the
  // PDF should be rasterized only for the physical display density. Including
  // the app font scale here would render an oversized bitmap and then make the
  // browser downsample it through the inverse document zoom, which softens text.
  const outputScale = Math.max(1, window.devicePixelRatio || 1)
  for (let pageNumber = 1; pageNumber <= pdf.numPages; pageNumber++) {
    const page = await pdf.getPage(pageNumber)
    const viewport = page.getViewport({ scale: zoom.value * 1.25 })
    const pageEl = document.createElement('section'); pageEl.className = 'pdf-page'; pageEl.dataset.page = String(pageNumber)
    pageEl.style.width = `${viewport.width}px`; pageEl.style.height = `${viewport.height}px`
    const canvas = document.createElement('canvas')
    canvas.width = Math.floor(viewport.width * outputScale)
    canvas.height = Math.floor(viewport.height * outputScale)
    canvas.style.width = `${viewport.width}px`
    canvas.style.height = `${viewport.height}px`
    const textLayer = document.createElement('div'); textLayer.className = 'pdf-text-layer'
    const context = canvas.getContext('2d'); if (!context) continue
    pageEl.append(canvas, textLayer); pdfEl.value.append(pageEl)
    await page.render({
      canvas,
      canvasContext: context,
      viewport,
      transform: outputScale === 1 ? undefined : [outputScale, 0, 0, outputScale, 0, 0],
    }).promise
    const content = await page.getTextContent()
    for (const raw of content.items as any[]) {
      if (!('str' in raw) || !raw.str) continue
      const tx = Util.transform(viewport.transform, raw.transform)
      const span = document.createElement('span'); span.textContent = raw.str; span.dataset.text = raw.str
      const fontHeight = Math.hypot(tx[2], tx[3])
      span.style.left = `${tx[4]}px`; span.style.top = `${tx[5] - fontHeight}px`; span.style.fontSize = `${fontHeight}px`
      textLayer.append(span)
    }
  }
}

async function loadDocx() {
  if (!sourceData || !docxEl.value) return
  const { renderAsync } = await import('docx-preview')
  docxEl.value.innerHTML = ''
  await renderAsync(new Blob([sourceData]), docxEl.value, undefined, { className: 'contract-docx', inWrapper: true, breakPages: true, ignoreLastRenderedPageBreak: true, useBase64URL: true })
  pageCount.value = Math.max(1, docxEl.value.querySelectorAll('section').length)
}

function clearMarks() {
  document.querySelectorAll('.review-text-mark').forEach((el) => { el.classList.remove('review-text-mark', 'review-text-mark--selected'); (el as HTMLElement).removeAttribute('data-issue-id') })
}

function applyIssueMarks() {
  clearMarks()
  for (const issue of props.issues || []) markIssue(issue)
  selectMark(props.selectedIssueId)
}

function markIssue(issue: ReviewIssue): HTMLElement | null {
  const needle = normalize(issue.original_quote)
  if (!needle) return null
  if (props.fileType === '.pdf' && pdfEl.value) {
    for (const page of Array.from(pdfEl.value.querySelectorAll<HTMLElement>('.pdf-page'))) {
      const spans = Array.from(page.querySelectorAll<HTMLElement>('.pdf-text-layer span'))
      const ranges: Array<{ span: HTMLElement; start: number; end: number }> = []
      let joined = ''
      for (const span of spans) {
        const text = normalize(span.dataset.text || '')
        if (!text) continue
        const start = joined.length; joined += text; ranges.push({ span, start, end: joined.length })
      }
      if (!hasReviewQuoteMatch(joined, needle)) continue
      let matchStart = joined.indexOf(needle); let matchLength = needle.length
      if (matchStart < 0) {
        const samples = [needle.slice(0, 40), needle.slice(Math.max(0, Math.floor(needle.length / 2) - 20), Math.floor(needle.length / 2) + 20), needle.slice(-40)]
        const sample = samples.find((part) => joined.includes(part)) || ''
        matchStart = joined.indexOf(sample); matchLength = sample.length
      }
      const matchEnd = matchStart + matchLength
      const targets = ranges.filter((range) => range.end > matchStart && range.start < matchEnd).map((range) => range.span)
      targets.forEach((span) => { span.classList.add('review-text-mark'); span.dataset.issueId = issue.id })
      return targets[0] || page
    }
  }
  if (props.fileType === '.docx' && docxEl.value) {
    const candidates = Array.from(docxEl.value.querySelectorAll<HTMLElement>('p, li, td'))
    const target = candidates.find((node) => hasReviewQuoteMatch(node.textContent || '', issue.original_quote))
    if (target) { target.classList.add('review-text-mark'); target.dataset.issueId = issue.id; return target }
  }
  return null
}

function selectMark(issueId?: string) {
  document.querySelectorAll('.review-text-mark--selected').forEach((el) => el.classList.remove('review-text-mark--selected'))
  if (!issueId) return
  const target = (props.fileType === '.pdf' ? pdfEl.value : docxEl.value)?.querySelector<HTMLElement>(`[data-issue-id="${CSS.escape(issueId)}"]`)
  target?.classList.add('review-text-mark--selected')
}

function locateIssue(issue: ReviewIssue) {
  const target = markIssue(issue)
  if (!target) { emit('locateFailed'); return false }
  selectMark(issue.id); target.scrollIntoView({ behavior: 'smooth', block: 'center' }); return true
}

function onDocumentClick(event: MouseEvent) {
  const target = (event.target as HTMLElement).closest<HTMLElement>('[data-issue-id]')
  if (target?.dataset.issueId) emit('markerClick', target.dataset.issueId)
}

function setZoom(value: number) {
  zoom.value = Math.min(2, Math.max(0.6, Number(value.toFixed(1))))
  if (props.fileType === '.pdf') void renderPdf().then(applyIssueMarks)
}

function goToPage(page: number) {
  const next = Math.min(pageCount.value || 1, Math.max(1, page)); currentPage.value = next
  const root = props.fileType === '.pdf' ? pdfEl.value : docxEl.value
  const pages = root?.querySelectorAll<HTMLElement>(props.fileType === '.pdf' ? '.pdf-page' : 'section')
  pages?.[next - 1]?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function updateCurrentPage() {
  const root = props.fileType === '.pdf' ? pdfEl.value : docxEl.value
  const pages = Array.from(root?.querySelectorAll<HTMLElement>(props.fileType === '.pdf' ? '.pdf-page' : 'section') || [])
  const top = scrollEl.value?.getBoundingClientRect().top || 0
  let best = 0, distance = Infinity
  pages.forEach((page, index) => { const next = Math.abs(page.getBoundingClientRect().top - top - 52); if (next < distance) { best = index; distance = next } })
  currentPage.value = best + 1
}

function observeAppFontScale() {
  if (typeof MutationObserver === 'undefined') return
  let previous = appFontScale()
  updateDocumentFontCompensation()
  fontScaleObserver = new MutationObserver(() => {
    const next = appFontScale()
    if (next === previous) return
    previous = next
    updateDocumentFontCompensation()
    if (pdf) void renderPdf().then(applyIssueMarks)
  })
  fontScaleObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['style'] })
}

watch(() => props.selectedIssueId, selectMark)
watch(() => props.issues, applyIssueMarks, { deep: true })
watch(() => props.reviewId, load)
onMounted(() => { observeAppFontScale(); void load() })
onBeforeUnmount(() => { fontScaleObserver?.disconnect(); void pdf?.destroy() })
defineExpose({ locateIssue, goToPage, setZoom })
</script>

<style scoped lang="less">
.document-viewer { height:100%; min-width:0; display:flex; flex-direction:column; background:#efefec; }
.document-viewer__toolbar { height:52px; padding:0 18px; display:flex; align-items:center; justify-content:space-between; background:#fff; border-bottom:1px solid #deded9; }
.document-viewer__file { min-width:0; display:flex; align-items:center; gap:8px; font-size:13px; font-weight:600; span { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; } }
.document-viewer__controls { display:flex; align-items:center; gap:7px; color:#5d5d58; font-size:12px; button { width:28px; height:28px; border:0; border-radius:5px; background:transparent; cursor:pointer; &:hover{background:#efefec;color:#111;} } i { width:1px;height:18px;background:#ddd; margin:0 4px; } }
.document-viewer__scroll { min-height:0; flex:1; overflow:auto; padding:28px; }
.document-viewer__state { height:100%; display:flex; align-items:center; justify-content:center; gap:10px; color:#73736d; &--error{color:#a53b31;} }
.document-viewer__pdf { min-width:max-content; display:flex; flex-direction:column; align-items:center; gap:22px; zoom:var(--document-font-compensation, 1); }
:deep(.pdf-page) { position:relative; flex:none; background:#fff; box-shadow:0 1px 6px rgba(0,0,0,.12); canvas{position:absolute;inset:0;} }
:deep(.pdf-text-layer) { position:absolute;inset:0;overflow:hidden;line-height:1; span{position:absolute;white-space:pre;transform-origin:0 0;color:transparent;cursor:text;} ::selection{background:rgba(88,111,94,.25);} }
.document-viewer__docx { width:max-content; min-width:100%; zoom:var(--document-font-compensation, 1); transform-origin:top center; :deep(.docx-wrapper){padding:0;background:transparent;} :deep(section){margin:0 auto 22px!important; transform:scale(var(--document-zoom)); transform-origin:top center; margin-bottom:calc((var(--document-zoom) - 1) * 1120px + 22px)!important;} }
:deep(.review-text-mark) { background:rgba(177,143,71,.24)!important; box-shadow:inset 3px 0 #b58b36; cursor:pointer!important; }
:deep(.review-text-mark--selected) { background:rgba(163,91,71,.28)!important; box-shadow:inset 3px 0 #8d4438; }
</style>
