<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useWebSocket } from '@/composables/useWebSocket'
import { getDoc, putDoc } from '@/api/docs'
import { ApiError } from '@/api/client'
import MarkdownEditor from '@/components/artifact/MarkdownEditor.vue'
import type { WsEvent } from '@/types/api'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const project = route.params.project as string
const pathMatch = route.params.pathMatch
const relPath = computed(() =>
  Array.isArray(pathMatch) ? pathMatch.join('/') : String(pathMatch),
)

// Breadcrumb: split into segments for display
const breadcrumbs = computed(() => ['docs', ...relPath.value.split('/')])

// Role-based edit permission: any role other than qa-only can edit
const canEdit = computed(() => {
  const roles = authStore.rolesForProject(project)
  return roles.length > 0 && roles.some((r) => r !== 'qa')
})

// Editor state
const body = ref('')
const fileSha = ref('')
const isMarkdown = ref(true)
const loadError = ref<string | null>(null)
const notFound = ref(false)
const saving = ref(false)

// Conflict / disk-change indicators
const conflictError = ref(false)
const diskUpdated = ref(false)
const isDirty = ref(false)

// Track initial body to detect dirty state
const savedBody = ref('')

async function loadDoc(): Promise<void> {
  loadError.value = null
  notFound.value = false
  try {
    const data = await getDoc(project, relPath.value)
    body.value = data.body ?? ''
    savedBody.value = data.body ?? ''
    fileSha.value = data.file_sha
    isMarkdown.value = data.is_markdown
    isDirty.value = false
    diskUpdated.value = false
    conflictError.value = false
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) {
      notFound.value = true
    } else {
      loadError.value = err instanceof Error ? err.message : 'Failed to load document'
    }
  }
}

async function save(): Promise<void> {
  if (saving.value || !canEdit.value) return
  saving.value = true
  conflictError.value = false
  try {
    const result = await putDoc(project, relPath.value, body.value, fileSha.value)
    fileSha.value = result.file_sha
    savedBody.value = body.value
    isDirty.value = false
  } catch (err) {
    if (err instanceof ApiError && err.status === 409) {
      conflictError.value = true
    } else {
      loadError.value = err instanceof Error ? err.message : 'Failed to save document'
    }
  } finally {
    saving.value = false
  }
}

function onBodyUpdate(val: string): void {
  body.value = val
  isDirty.value = val !== savedBody.value
  diskUpdated.value = false
}

async function reload(): Promise<void> {
  await loadDoc()
}

function goBack(): void {
  router.push({ name: 'docs', params: { project } })
}

onMounted(() => {
  loadDoc()
})

useWebSocket(project, 'doc.changed', (e: WsEvent) => {
  const changedPath = e.payload.path as string
  if (changedPath !== relPath.value) return
  if (isDirty.value) {
    // User has unsaved changes — show non-blocking notice
    diskUpdated.value = true
  } else {
    // No local changes — silently pull the latest version
    loadDoc()
  }
})

function rawDownloadUrl(): string {
  return `/api/p/${project}/docs/${relPath.value.split('/').map(encodeURIComponent).join('/')}`
}
</script>

<template>
  <div class="docs-editor-view">
    <!-- Header / breadcrumb -->
    <div class="editor-header">
      <button class="btn-back" @click="goBack">← Back to documents</button>
      <nav class="breadcrumb" aria-label="File path">
        <span
          v-for="(seg, i) in breadcrumbs"
          :key="i"
          class="breadcrumb-seg"
        >
          <span v-if="i > 0" class="breadcrumb-sep">/</span>
          {{ seg }}
        </span>
      </nav>
    </div>

    <!-- 404 -->
    <div v-if="notFound" class="empty-state">
      <p>Document not found — it may have been removed.</p>
      <button class="btn-action" @click="goBack">Back to documents list</button>
    </div>

    <!-- Load error -->
    <div v-else-if="loadError" class="error-banner">
      {{ loadError }}
    </div>

    <!-- Non-markdown fallback -->
    <div v-else-if="!isMarkdown" class="non-markdown-panel">
      <p class="non-markdown-name">{{ relPath }}</p>
      <p class="non-markdown-msg">This file type can't be edited inline.</p>
      <a :href="rawDownloadUrl()" class="btn-action" download>Download file</a>
    </div>

    <!-- Markdown editor -->
    <template v-else>
      <!-- Conflict banner -->
      <div v-if="conflictError" class="conflict-banner" role="alert">
        Document was modified on disk — reload to see latest.
        <button class="btn-inline" @click="reload">Reload</button>
      </div>

      <!-- Disk-updated indicator (non-blocking) -->
      <div v-if="diskUpdated" class="disk-updated-banner" role="status">
        Disk version updated while you were editing.
        <button class="btn-inline" @click="reload">Reload latest</button>
        <button class="btn-inline" @click="diskUpdated = false">Dismiss</button>
      </div>

      <div class="editor-toolbar">
        <button
          v-if="canEdit"
          class="btn-save"
          :disabled="saving || !isDirty"
          @click="save"
        >
          {{ saving ? 'Saving…' : 'Save' }}
        </button>
        <span v-else class="read-only-badge">Read-only</span>
      </div>

      <div class="editor-body">
        <MarkdownEditor
          :model-value="body"
          :readonly="!canEdit"
          @update:model-value="onBodyUpdate"
          @save="save"
        />
      </div>
    </template>
  </div>
</template>

<style scoped>
.docs-editor-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.editor-header {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.btn-back {
  background: none;
  border: none;
  color: var(--color-accent);
  cursor: pointer;
  font-size: var(--text-sm);
  padding: 0;
  white-space: nowrap;
}

.btn-back:hover {
  text-decoration: underline;
}

.breadcrumb {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  font-family: var(--font-mono, monospace);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.breadcrumb-sep {
  margin: 0 2px;
  opacity: 0.5;
}

.empty-state,
.non-markdown-panel {
  padding: var(--space-8);
  text-align: center;
  color: var(--color-text-muted);
}

.non-markdown-name {
  font-family: var(--font-mono, monospace);
  margin-bottom: var(--space-2);
}

.non-markdown-msg {
  margin-bottom: var(--space-4);
}

.btn-action {
  display: inline-block;
  padding: var(--space-2) var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: none;
  color: var(--color-text);
  font-size: var(--text-sm);
  cursor: pointer;
  text-decoration: none;
  transition: background 0.1s;
}

.btn-action:hover {
  background: var(--color-border);
}

.conflict-banner,
.disk-updated-banner {
  padding: var(--space-2) var(--space-4);
  font-size: var(--text-sm);
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-shrink: 0;
}

.conflict-banner {
  background: var(--color-warning-bg, #fef3c7);
  color: var(--color-warning-text, #92400e);
  border-bottom: 1px solid var(--color-warning-border, #fcd34d);
}

.disk-updated-banner {
  background: var(--color-info-bg, #eff6ff);
  color: var(--color-info-text, #1e40af);
  border-bottom: 1px solid var(--color-info-border, #bfdbfe);
}

.btn-inline {
  background: none;
  border: 1px solid currentColor;
  border-radius: var(--radius-sm);
  padding: 2px 8px;
  font-size: var(--text-xs);
  cursor: pointer;
  color: inherit;
}

.error-banner {
  padding: var(--space-3) var(--space-4);
  background: var(--color-error-bg, #fee2e2);
  color: var(--color-error-text, #991b1b);
  font-size: var(--text-sm);
  flex-shrink: 0;
}

.editor-toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-4);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.btn-save {
  padding: var(--space-1) var(--space-4);
  background: var(--color-accent);
  border: none;
  border-radius: var(--radius-md);
  color: #fff;
  font-size: var(--text-sm);
  cursor: pointer;
  transition: opacity 0.1s;
}

.btn-save:disabled {
  opacity: 0.45;
  cursor: default;
}

.read-only-badge {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
}

.editor-body {
  flex: 1;
  overflow: hidden;
}
</style>
