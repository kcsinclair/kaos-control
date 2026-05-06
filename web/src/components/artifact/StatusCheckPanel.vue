<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { checkStatus, advanceStatuses } from '@/api/statusCheck'
import type { StaleArtifact, StatusCheckResponse } from '@/api/statusCheck'
import { useWebSocket } from '@/composables/useWebSocket'

const props = defineProps<{
  project: string
  lineage?: string
}>()

const emit = defineEmits<{ close: [] }>()

const loading = ref(false)
const error = ref<string | null>(null)
const results = ref<StatusCheckResponse | null>(null)
const advancingPaths = ref<Set<string>>(new Set())
const fixAllLoading = ref(false)

const groupedByLineage = computed<Record<string, StaleArtifact[]>>(() => {
  if (!results.value) return {}
  const groups: Record<string, StaleArtifact[]> = {}
  for (const item of results.value.stale) {
    if (!groups[item.lineage]) groups[item.lineage] = []
    groups[item.lineage].push(item)
  }
  return groups
})

const hasAdvanceable = computed(() =>
  results.value?.stale.some((a) => a.can_advance) ?? false
)

async function fetchResults() {
  loading.value = true
  error.value = null
  try {
    results.value = await checkStatus(props.project, props.lineage)
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : 'Failed to check status'
  } finally {
    loading.value = false
  }
}

async function advance(artifact: StaleArtifact) {
  const next = new Set(advancingPaths.value)
  next.add(artifact.path)
  advancingPaths.value = next
  error.value = null
  try {
    await advanceStatuses(props.project, [artifact.path])
    await fetchResults()
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : 'Failed to advance'
  } finally {
    const after = new Set(advancingPaths.value)
    after.delete(artifact.path)
    advancingPaths.value = after
  }
}

async function fixAll() {
  if (!results.value) return
  const paths = results.value.stale.filter((a) => a.can_advance).map((a) => a.path)
  if (!paths.length) return
  fixAllLoading.value = true
  error.value = null
  try {
    await advanceStatuses(props.project, paths)
    await fetchResults()
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : 'Failed to fix all'
  } finally {
    fixAllLoading.value = false
  }
}

function displayPath(path: string): string {
  return path.split('/').pop() ?? path
}

// Debounced WS refresh — handled in Milestone 5; stub here for structure
let debounceTimer: ReturnType<typeof setTimeout> | null = null

function scheduleRefresh() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    fetchResults()
  }, 500)
}

onUnmounted(() => {
  if (debounceTimer) clearTimeout(debounceTimer)
})

useWebSocket(props.project, 'artifact.indexed', () => {
  scheduleRefresh()
})

watch(() => [props.project, props.lineage] as const, fetchResults, { immediate: true })
</script>

<template>
  <div class="sc-panel">
    <div class="sc-panel-header">
      <span class="sc-panel-title">
        {{ lineage ? `Status: ${lineage}` : 'Status: all lineages' }}
      </span>
      <div class="sc-header-actions">
        <button
          v-if="hasAdvanceable && !loading"
          class="sc-btn sc-btn--primary"
          :disabled="fixAllLoading"
          @click="fixAll"
        >
          <span v-if="fixAllLoading" class="sc-spinner" aria-hidden="true"></span>
          {{ fixAllLoading ? 'Fixing…' : 'Fix all' }}
        </button>
        <button class="sc-close" aria-label="Close status panel" @click="emit('close')">✕</button>
      </div>
    </div>

    <div v-if="error" class="sc-error">{{ error }}</div>

    <div v-if="loading" class="sc-state">
      <span class="sc-spinner sc-spinner--lg" aria-hidden="true"></span>
      Checking statuses…
    </div>

    <div
      v-else-if="results && results.stale.length === 0"
      class="sc-state sc-state--empty"
    >
      No stale statuses found
    </div>

    <div v-else-if="results" class="sc-results">
      <div
        v-for="(artifacts, lin) in groupedByLineage"
        :key="lin"
        class="sc-lineage-group"
      >
        <div class="sc-lineage-label">{{ lin }}</div>
        <div
          v-for="artifact in artifacts"
          :key="artifact.path"
          class="sc-artifact-row"
        >
          <div class="sc-artifact-info">
            <span class="sc-artifact-title">
              {{ artifact.title || displayPath(artifact.path) }}
            </span>
            <div class="sc-artifact-badges">
              <span class="sc-badge sc-badge--current">{{ artifact.current_status }}</span>
              <span class="sc-arrow" aria-hidden="true">→</span>
              <span class="sc-badge sc-badge--suggested">{{ artifact.suggested_status }}</span>
            </div>
            <div v-if="artifact.children.length" class="sc-children">
              <span class="sc-children-label">Because:</span>
              <span
                v-for="child in artifact.children"
                :key="child.path"
                class="sc-child-item"
              >{{ displayPath(child.path) }} ({{ child.status }})</span>
            </div>
          </div>
          <button
            class="sc-btn sc-btn--advance"
            :disabled="!artifact.can_advance || advancingPaths.has(artifact.path)"
            :title="artifact.can_advance ? `Advance to ${artifact.suggested_status}` : artifact.blocked_reason"
            @click="artifact.can_advance && advance(artifact)"
          >
            <span
              v-if="advancingPaths.has(artifact.path)"
              class="sc-spinner"
              aria-hidden="true"
            ></span>
            {{ advancingPaths.has(artifact.path) ? '…' : 'Advance' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sc-panel {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.sc-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
  gap: var(--space-3);
}
.sc-panel-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}
.sc-header-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.sc-close {
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: var(--text-base);
  line-height: 1;
  padding: 2px;
}
.sc-close:hover { color: var(--color-text); }
.sc-error {
  padding: var(--space-3) var(--space-4);
  font-size: var(--text-sm);
  color: #dc2626;
}
.sc-state {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  justify-content: center;
  padding: var(--space-6) var(--space-4);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.sc-state--empty {
  color: var(--color-text-muted);
}
.sc-results {
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
}
.sc-lineage-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.sc-lineage-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
}
.sc-artifact-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}
.sc-artifact-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  flex: 1;
}
.sc-artifact-title {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.sc-artifact-badges {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}
.sc-badge {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
}
.sc-badge--current {
  background: #e5e7eb;
  color: #374151;
}
.sc-badge--suggested {
  background: #dbeafe;
  color: #1e40af;
}
.sc-arrow {
  font-size: 11px;
  color: var(--color-text-muted);
}
.sc-children {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--color-text-muted);
}
.sc-children-label {
  font-weight: 500;
}
.sc-child-item {
  font-family: monospace;
}
.sc-btn {
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  white-space: nowrap;
  flex-shrink: 0;
}
.sc-btn:hover:not(:disabled) { background: var(--color-border); }
.sc-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.sc-btn--primary {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}
.sc-btn--primary:hover:not(:disabled) { opacity: 0.88; background: var(--color-accent); }
.sc-btn--advance {
  font-size: 11px;
  padding: 2px var(--space-3);
}
.sc-spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: sc-spin 0.6s linear infinite;
}
.sc-spinner--lg {
  width: 16px;
  height: 16px;
}
@keyframes sc-spin {
  to { transform: rotate(360deg); }
}
@media (prefers-reduced-motion: reduce) {
  .sc-spinner { animation: none; }
}
</style>
