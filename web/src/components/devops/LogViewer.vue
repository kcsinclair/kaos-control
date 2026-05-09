<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useDevOpsStore } from '@/stores/devops'
import { useUiStore } from '@/stores/ui'

const props = defineProps<{
  project: string
  runId: string
  pipelineName: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const devops = useDevOpsStore()
const ui = useUiStore()

const loading = ref(true)
const logContent = ref('')

onMounted(async () => {
  try {
    logContent.value = await devops.fetchRunLog(props.project, props.runId)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to load run log.')
    logContent.value = '(failed to load log)'
  } finally {
    loading.value = false
  }
})

function formatLogLine(line: string): { type: string; text: string } {
  try {
    const obj = JSON.parse(line)
    return { type: obj.type ?? 'raw', text: JSON.stringify(obj, null, 2) }
  } catch {
    return { type: 'raw', text: line }
  }
}

function parsedLines() {
  if (!logContent.value) return []
  return logContent.value
    .split('\n')
    .filter((l) => l.trim().length > 0)
    .map(formatLogLine)
}
</script>

<template>
  <div class="log-viewer-backdrop" @click.self="emit('close')">
    <div class="log-viewer">
      <div class="log-viewer__header">
        <span class="log-viewer__title">Run log — {{ props.pipelineName }}</span>
        <span class="log-viewer__run-id">{{ props.runId }}</span>
        <button class="log-viewer__close" @click="emit('close')">✕</button>
      </div>
      <div class="log-viewer__body">
        <div v-if="loading" class="log-loading">Loading…</div>
        <pre v-else class="log-pre"><template
          v-for="(line, i) in parsedLines()"
          :key="i"
        ><span
            class="log-line"
            :class="`log-line--${line.type}`"
          >{{ line.text }}</span>
</template></pre>
      </div>
    </div>
  </div>
</template>

<style scoped>
.log-viewer-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}
.log-viewer {
  background: #0f172a;
  border: 1px solid var(--color-border-dark);
  border-radius: var(--radius-md);
  width: min(900px, 92vw);
  height: min(80vh, 700px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.log-viewer__header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  flex-shrink: 0;
}
.log-viewer__title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: #e2e8f0;
  flex: 1;
}
.log-viewer__run-id {
  font-family: monospace;
  font-size: 11px;
  color: #64748b;
}
.log-viewer__close {
  background: none;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0;
  flex-shrink: 0;
}
.log-viewer__close:hover { color: #e2e8f0; }
.log-viewer__body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-3) var(--space-4);
}
.log-loading {
  color: #94a3b8;
  font-size: var(--text-sm);
}
.log-pre {
  margin: 0;
  font-family: monospace;
  font-size: 12px;
  color: #e2e8f0;
  white-space: pre-wrap;
  word-break: break-all;
  line-height: 1.6;
}
.log-line {
  display: block;
}
.log-line--pipeline\.run\.started,
.log-line--pipeline\.step\.started {
  color: #93c5fd;
}
.log-line--pipeline\.step\.output {
  color: #e2e8f0;
}
.log-line--pipeline\.step\.completed {
  color: #86efac;
}
.log-line--pipeline\.run\.completed {
  color: #fde68a;
  font-weight: 600;
}
.log-line--raw {
  color: #94a3b8;
}
</style>
