<template>
  <div class="legal-workspace-shell" :class="{ 'legal-workspace-shell--collapsed': collapsed }">
    <LegalWorkspaceSidebar :collapsed="collapsed" @toggle="toggleSidebar" />
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
  document.addEventListener('dragenter', onDragEnter, true)
  document.addEventListener('dragover', onDragOver, true)
  document.addEventListener('dragleave', onDragLeave, true)
  document.addEventListener('drop', onDrop, true)
})

onUnmounted(() => {
  document.removeEventListener('dragenter', onDragEnter, true)
  document.removeEventListener('dragover', onDragOver, true)
  document.removeEventListener('dragleave', onDragLeave, true)
  document.removeEventListener('drop', onDrop, true)
})
</script>

<style lang="less">
.legal-workspace-shell {
  --td-brand-color-light: #efefec;
  --td-brand-color-focus: rgba(17, 17, 17, 0.14);
  --td-brand-color-disabled: #aaa9a3;
  --td-brand-color-hover: #30302d;
  --td-brand-color: #171715;
  --td-brand-color-active: #000;
  --td-bg-color-page: #f4f4f1;
  --td-bg-color-sidebar: #fff;
  --td-bg-color-container: #fff;
  --td-bg-color-container-select: #fff;
  --td-bg-color-container-hover: #f4f4f1;
  --td-bg-color-container-active: #e8e8e4;
  --td-bg-color-secondarycontainer: #f2f2ef;
  --td-bg-color-secondarycontainer-hover: #e9e9e5;
  --td-bg-color-component: #e8e8e4;
  --td-bg-color-component-hover: #deded9;
  --td-bg-color-component-disabled: #efefec;
  --td-component-stroke: #e4e4df;
  --td-component-border: #d7d7d1;
  --td-font-gray-1: rgba(17, 17, 17, 0.92);
  --td-font-gray-2: rgba(17, 17, 17, 0.62);
  --td-font-gray-3: rgba(17, 17, 17, 0.42);
  --td-font-gray-4: rgba(17, 17, 17, 0.28);
  --td-text-color-primary: var(--td-font-gray-1);
  --td-text-color-secondary: var(--td-font-gray-2);
  --td-text-color-placeholder: var(--td-font-gray-3);
  --td-text-color-disabled: var(--td-font-gray-4);

  width: 100%;
  height: 100%;
  min-width: 720px;
  min-height: 0;
  display: flex;
  overflow: hidden;
  color-scheme: light;
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-page);
}

.legal-workspace-outlet {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--td-bg-color-container);
}

.legal-workspace-drop-mask {
  position: fixed;
  inset: 0;
  z-index: 999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.84);
}
</style>
