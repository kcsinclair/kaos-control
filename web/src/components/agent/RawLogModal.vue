<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import * as agentsApi from '@/api/agents'

const props = defineProps<{
  project: string
  runId: string
}>()

const emit = defineEmits<{ close: [] }>()

const logContent = ref<string | null>(null)
const loading = ref(true)
const fetchError = ref<string | null>(null)

let previousFocus: HTMLElement | null = null

onMounted(async () => {
  previousFocus = document.activeElement as HTMLElement | null
  try {
    logContent.value = await agentsApi.getRunLog(props.project, props.runId)
  } catch (e: unknown) {
    fetchError.value = e instanceof Error ? e.message : 'Failed to load log'
  } finally {
    loading.value = false
  }
})

onBeforeUnmount(() => {
  previousFocus?.focus()
})

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close')
    return
  }
  if (e.key === 'Tab') {
    const panel = (e.currentTarget as HTMLElement).querySelector<HTMLElement>('.rlm-panel')
    if (!panel) return
    const focusable = Array.from(
      panel.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
      ),
    ).filter((el) => !el.hasAttribute('disabled'))
    if (!focusable.length) { e.preventDefault(); return }
    const first = focusable[0]
    const last = focusable[focusable.length - 1]
    if (e.shiftKey) {
      if (document.activeElement === first) { e.preventDefault(); last.focus() }
    } else {
      if (document.activeElement === last) { e.preventDefault(); first.focus() }
    }
  }
}

function handleOverlayClick(e: MouseEvent) {
  if ((e.target as HTMLElement).classList.contains('rlm-overlay')) emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div
      class="rlm-overlay"
      role="dialog"
      aria-modal="true"
      aria-label="Raw log"
      tabindex="-1"
      @click="handleOverlayClick"
      @keydown="handleKeydown"
    >
      <div class="rlm-panel">
        <div class="rlm-header">
          <h3 class="rlm-title">Raw Log</h3>
          <button class="rlm-close" aria-label="Close log" @click="emit('close')">✕</button>
        </div>

        <div v-if="loading" class="rlm-state">Loading log…</div>
        <div v-else-if="fetchError" class="rlm-state rlm-state--error">{{ fetchError }}</div>
        <div v-else-if="!logContent || logContent.length === 0" class="rlm-state">
          No log content available.
        </div>
        <pre v-else class="rlm-content">{{ logContent }}</pre>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.rlm-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  display: flex;
  align-items: stretch;
  justify-content: center;
  z-index: 310;
  padding: var(--space-4);
}

.rlm-panel {
  position: relative;
  background: #0f172a;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 100%;
  max-width: 900px;
  min-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.rlm-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6) var(--space-3);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  flex-shrink: 0;
}

.rlm-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: #e2e8f0;
}

.rlm-close {
  background: none;
  border: none;
  font-size: var(--text-lg);
  color: #94a3b8;
  cursor: pointer;
  line-height: 1;
  padding: var(--space-1);
}
.rlm-close:hover { color: #e2e8f0; }

.rlm-state {
  padding: var(--space-6);
  font-size: var(--text-sm);
  color: #94a3b8;
}

.rlm-state--error { color: #fca5a5; }

.rlm-content {
  flex: 1;
  margin: 0;
  padding: var(--space-4) var(--space-6);
  font-family: monospace;
  font-size: 12px;
  color: #e2e8f0;
  white-space: pre-wrap;
  word-break: break-all;
  overflow-y: auto;
  line-height: 1.6;
}
</style>
