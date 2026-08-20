<template>
  <div class="legal-workspace-shell" :class="{ 'legal-workspace-shell--collapsed': collapsed }">
    <LegalWorkspaceSidebar :collapsed="collapsed" @toggle="toggleSidebar" @expand="expandSidebar" />
    <main class="legal-workspace-outlet">
      <RouterView />
    </main>
    <div v-show="dropMaskVisible" class="legal-workspace-drop-mask">
      <UploadMask />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

import UploadMask from '@/components/upload-mask.vue'
import {
  LEGAL_ASSISTANT_CHAT_ROUTE,
  LEGAL_ASSISTANT_HOME_ROUTE,
} from '@/router/paths'
import LegalWorkspaceSidebar from './LegalWorkspaceSidebar.vue'

const SIDEBAR_STORAGE_KEY = 'lexai_legal_sidebar_collapsed'
const WORKSPACE_THEME_ATTRIBUTE = 'data-workspace-theme'
const route = useRoute()
const collapsed = ref(localStorage.getItem(SIDEBAR_STORAGE_KEY) === 'true')
const dropMaskVisible = ref(false)
let dragCounter = 0

const assistantRouteNames = new Set([LEGAL_ASSISTANT_HOME_ROUTE, LEGAL_ASSISTANT_CHAT_ROUTE])
const isAssistantRoute = () => assistantRouteNames.has(String(route.name || ''))

const toggleSidebar = () => {
  collapsed.value = !collapsed.value
  localStorage.setItem(SIDEBAR_STORAGE_KEY, String(collapsed.value))
}

const expandSidebar = () => {
  if (!collapsed.value) return
  collapsed.value = false
  localStorage.setItem(SIDEBAR_STORAGE_KEY, 'false')
}

const isFileDrag = (event: DragEvent) => Array.from(event.dataTransfer?.types || []).includes('Files')

const collectDroppedFiles = async (event: DragEvent): Promise<File[]> => {
  const directFiles = event.dataTransfer?.files ? Array.from(event.dataTransfer.files) : []
  if (directFiles.length > 0) return directFiles

  const items = event.dataTransfer?.items ? Array.from(event.dataTransfer.items) : []
  const files = await Promise.all(items.map((item) => new Promise<File | null>((resolve) => {
    const entry = (item as any).webkitGetAsEntry?.() as {
      isFile?: boolean
      file?: (success: (file: File) => void, failure: () => void) => void
    } | null | undefined
    if (entry?.isFile && typeof entry.file === 'function') {
      entry.file((file: File) => resolve(file), () => resolve(null))
      return
    }
    resolve(null)
  })))
  return files.filter((file): file is File => file instanceof File)
}

const onDragEnter = (event: DragEvent) => {
  if (!isAssistantRoute() || !isFileDrag(event)) return
  event.preventDefault()
  dragCounter += 1
  dropMaskVisible.value = true
}

const onDragOver = (event: DragEvent) => {
  if (!isAssistantRoute() || !isFileDrag(event)) return
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
}

const onDragLeave = (event: DragEvent) => {
  if (!isAssistantRoute() || !isFileDrag(event)) return
  event.preventDefault()
  dragCounter = Math.max(0, dragCounter - 1)
  if (dragCounter === 0) dropMaskVisible.value = false
}

const onDrop = async (event: DragEvent) => {
  if (!isAssistantRoute() || !isFileDrag(event)) return
  event.preventDefault()
  event.stopPropagation()
  dragCounter = 0
  dropMaskVisible.value = false
  const files = await collectDroppedFiles(event)
  if (files.length === 0) return
  window.dispatchEvent(new CustomEvent('weknora:chat-file-drop', { detail: { files } }))
}

watch(() => route.name, () => {
  dragCounter = 0
  dropMaskVisible.value = false
})

onMounted(() => {
  document.documentElement.setAttribute(WORKSPACE_THEME_ATTRIBUTE, 'legal')
  document.addEventListener('dragenter', onDragEnter, true)
  document.addEventListener('dragover', onDragOver, true)
  document.addEventListener('dragleave', onDragLeave, true)
  document.addEventListener('drop', onDrop, true)
})

onUnmounted(() => {
  if (document.documentElement.getAttribute(WORKSPACE_THEME_ATTRIBUTE) === 'legal') {
    document.documentElement.removeAttribute(WORKSPACE_THEME_ATTRIBUTE)
  }
  document.removeEventListener('dragenter', onDragEnter, true)
  document.removeEventListener('dragover', onDragOver, true)
  document.removeEventListener('dragleave', onDragLeave, true)
  document.removeEventListener('drop', onDrop, true)
})
</script>

<style lang="less">
.legal-workspace-shell,
:root:root[data-workspace-theme='legal'] {
  --legal-bg-page: #f7f4ed;
  --legal-bg-surface: #fcfbf7;
  --legal-bg-paper: #fff;
  --legal-bg-hover: #f0ede6;
  --legal-bg-active: #e9e9e7;
  --legal-text-primary: #1f1f1f;
  --legal-text-secondary: #6b6b6b;
  --legal-text-disabled: #a0a0a0;
  --legal-brand: #1f1f1f;
  --legal-brand-hover: #2a2a2a;
  --legal-brand-active: #171717;
  --legal-ai: #737373;
  --legal-ai-strong: #4d4d4d;
  --legal-ai-soft: #f1f1ef;
  --legal-border: #e2ded6;
  --legal-border-strong: #d5d0c6;
  --legal-warning: #a9793d;
  --legal-warning-strong: #765329;
  --legal-warning-soft: #f4ebdd;
  --legal-risk: #a6534d;
  --legal-risk-strong: #87423d;
  --legal-risk-soft: #f4e7e4;
  --legal-status-queued: #0072f5;
  --legal-status-queued-strong: #0059b8;
  --legal-status-queued-soft: #eaf4ff;
  --legal-status-running: #f5a623;
  --legal-status-running-strong: #9a5b00;
  --legal-status-running-soft: #fff3d6;
  --legal-status-completed: #0cce6b;
  --legal-status-completed-strong: #087a43;
  --legal-status-completed-soft: #e6f9ef;
  --legal-status-failed: #ee0000;
  --legal-status-failed-strong: #b00000;
  --legal-status-failed-soft: #ffebeb;
  --legal-status-cancelled: #8f8f8f;
  --legal-status-cancelled-strong: #666;
  --legal-status-cancelled-soft: #f1f1f1;
  --legal-status-review: #8e4ec6;
  --legal-status-review-strong: #663399;
  --legal-status-review-soft: #f3eefe;
  --legal-focus-ring: rgba(31, 31, 31, 0.18);
  --legal-overlay: rgba(31, 31, 31, 0.24);
  --legal-shadow-soft: 0 8px 24px rgba(31, 31, 31, 0.07);

  --td-brand-color-1: var(--legal-ai-soft);
  --td-brand-color-2: #dededb;
  --td-brand-color-3: #a8a8a5;
  --td-brand-color-4: var(--legal-brand);
  --td-brand-color-5: var(--legal-brand-hover);
  --td-brand-color-6: var(--legal-brand-active);
  --td-brand-color-light: var(--legal-ai-soft);
  --td-brand-color-focus: var(--legal-focus-ring);
  --td-brand-color-disabled: #a3a3a0;
  --td-brand-color-hover: var(--legal-brand-hover);
  --td-brand-color: var(--legal-brand);
  --td-brand-color-active: var(--legal-brand-active);
  --td-warning-color-1: var(--legal-warning-soft);
  --td-warning-color-2: #e7d2b2;
  --td-warning-color-5: var(--legal-warning);
  --td-warning-color-6: var(--legal-warning-strong);
  --td-warning-color-light: var(--legal-warning-soft);
  --td-warning-color: var(--legal-warning-strong);
  --td-warning-color-hover: #b98a4d;
  --td-warning-color-active: #8f642f;
  --td-error-color-1: var(--legal-risk-soft);
  --td-error-color-2: #dfb7b3;
  --td-error-color-5: var(--legal-risk);
  --td-error-color-6: var(--legal-risk-strong);
  --td-error-color-light: var(--legal-risk-soft);
  --td-error-color: var(--legal-risk-strong);
  --td-error-color-hover: #b76660;
  --td-error-color-active: #8e453f;
  --td-success-color-1: var(--legal-ai-soft);
  --td-success-color-2: #dededb;
  --td-success-color-5: var(--legal-ai);
  --td-success-color-6: var(--legal-ai-strong);
  --td-success-color-light: var(--legal-ai-soft);
  --td-success-color: var(--legal-ai-strong);
  --td-success-color-hover: var(--legal-ai);
  --td-success-color-active: #3f3f3f;
  --td-bg-color-page: var(--legal-bg-page);
  --td-bg-color-sidebar: var(--legal-bg-surface);
  --td-bg-color-container: var(--legal-bg-surface);
  --td-bg-color-container-select: var(--legal-bg-surface);
  --td-bg-color-container-hover: var(--legal-bg-hover);
  --td-bg-color-container-active: var(--legal-bg-active);
  --td-bg-color-secondarycontainer: #f3f0e9;
  --td-bg-color-secondarycontainer-hover: var(--legal-bg-hover);
  --td-bg-color-secondarycontainer-active: var(--legal-bg-active);
  --td-bg-color-component: #ebe8e1;
  --td-bg-color-component-hover: #e3dfd6;
  --td-bg-color-component-active: #d6d2c9;
  --td-bg-color-component-disabled: #f1eee7;
  --td-component-stroke: var(--legal-border);
  --td-component-border: var(--legal-border);
  --td-border-level-1-color: var(--legal-border);
  --td-border-level-2-color: var(--legal-border-strong);
  --td-font-gray-1: var(--legal-text-primary);
  --td-font-gray-2: var(--legal-text-secondary);
  --td-font-gray-3: #828282;
  --td-font-gray-4: var(--legal-text-disabled);
  --td-text-color-primary: var(--td-font-gray-1);
  --td-text-color-secondary: var(--td-font-gray-2);
  --td-text-color-placeholder: var(--td-font-gray-3);
  --td-text-color-disabled: var(--td-font-gray-4);
  --td-text-color-brand: var(--legal-ai-strong);
  --td-text-color-link: var(--legal-ai-strong);
  --td-text-color-anti: #fff;
  --td-shadow-1: var(--legal-shadow-soft);
  --td-shadow-2: 0 12px 32px rgba(31, 31, 31, 0.1);
  --weknora-faq-color: var(--legal-ai-strong);
  --attachment-icon-shell: var(--legal-brand);
  --attachment-icon-body: var(--legal-ai);
  --attachment-icon-fold: var(--legal-ai-strong);
  color-scheme: light;
}

:root:root[data-workspace-theme='legal'] {
  background: var(--legal-bg-page);
}

:root[data-workspace-theme='legal'] .t-popup__content,
:root[data-workspace-theme='legal'] .t-select__dropdown,
:root[data-workspace-theme='legal'] .t-dropdown__menu,
:root[data-workspace-theme='legal'] .t-message,
:root[data-workspace-theme='legal'] .t-notification {
  color: var(--legal-text-primary);
  background: var(--legal-bg-surface) !important;
  border-color: var(--legal-border) !important;
}

:root[theme-mode='dark'][data-workspace-theme='legal'] .chat-header-menu-popup .t-popup__content {
  color: var(--legal-text-primary) !important;
  background: var(--legal-bg-surface) !important;
  border-color: var(--legal-border) !important;
  box-shadow: var(--legal-shadow-soft) !important;
}

:root[data-workspace-theme='legal'] :is(button, a, input, select, textarea, [tabindex]):focus-visible {
  outline: 2px solid var(--legal-ai);
  outline-offset: 2px;
}

/* The legal assistant input stays visually quiet while focused; the caret
   remains visible and keyboard behavior is unchanged. */
:root[data-workspace-theme='legal'] .rich-input-container:focus-within {
  border-color: var(--legal-border) !important;
  box-shadow: none !important;
}

:root[data-workspace-theme='legal'] .rich-input-container :is(input, textarea, button):focus-visible {
  outline: none !important;
}

.legal-workspace-shell {
  width: 100%;
  height: 100%;
  min-width: 720px;
  min-height: 0;
  display: flex;
  overflow: hidden;
  color: var(--legal-text-primary);
  background: var(--legal-bg-page);
}

.legal-workspace-outlet {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--legal-bg-surface);
}

.legal-workspace-drop-mask {
  position: fixed;
  inset: 0;
  z-index: 999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(247, 244, 237, 0.88);
  backdrop-filter: blur(2px);
}
</style>
