<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import type { ArtifactRow } from '@/types/api'

const props = defineProps<{
  project: string
  artifacts: ArtifactRow[]
}>()

const emit = defineEmits<{
  openArtifact: [path: string]
}>()

// ── Collapse state (persisted in sessionStorage) ────────────────────────────

const SESSION_KEY = 'backlog-panel-collapsed'

const collapsed = ref(true)

onMounted(() => {
  const stored = sessionStorage.getItem(SESSION_KEY)
  if (stored !== null) {
    collapsed.value = stored === 'true'
  }
})

watch(collapsed, (v) => {
  sessionStorage.setItem(SESSION_KEY, String(v))
})

function toggleCollapsed() {
  collapsed.value = !collapsed.value
}

// ── Filter state ────────────────────────────────────────────────────────────

const filterType = ref('')
const filterStatus = ref('')
const filterPriority = ref('')

// Derive unique option values from the artifact list
const typeOptions = computed(() => {
  const types = new Set(props.artifacts.map((a) => a.type))
  return [...types].sort()
})

const statusOptions = computed(() => {
  const statuses = new Set(props.artifacts.map((a) => a.status))
  return [...statuses].sort()
})

const priorityOptions = computed(() => {
  const priorities = new Set(
    props.artifacts.flatMap((a) => (a.frontmatter?.priority ? [a.frontmatter.priority] : []))
  )
  return [...priorities].sort()
})

// ── Filtered list ───────────────────────────────────────────────────────────

const filtered = computed(() => {
  return props.artifacts.filter((a) => {
    if (filterType.value && a.type !== filterType.value) return false
    if (filterStatus.value && a.status !== filterStatus.value) return false
    if (filterPriority.value && a.frontmatter?.priority !== filterPriority.value) return false
    return true
  })
})

// ── Styling helpers ─────────────────────────────────────────────────────────

const TYPE_COLORS: Record<string, string> = {
  idea:           '#8b5cf6',
  ticket:         '#3b82f6',
  epic:           '#6366f1',
  'plan-backend': '#0891b2',
  'plan-frontend':'#0284c7',
  'plan-dev':     '#0369a1',
  'plan-test':    '#0e7490',
  test:           '#16a34a',
  prototype:      '#d97706',
  release:        '#dc2626',
  sprint:         '#7c3aed',
  defect:         '#b91c1c',
}

function typeBadgeColor(type: string): string {
  return TYPE_COLORS[type] ?? '#64748b'
}

const STATUS_BORDER_COLORS: Record<string, string> = {
  draft:          '#94a3b8',
  clarifying:     '#60a5fa',
  planning:       '#818cf8',
  'in-development': '#f59e0b',
  'in-qa':        '#fb923c',
  approved:       '#34d399',
  rejected:       '#f87171',
  abandoned:      '#94a3b8',
  done:           '#4ade80',
  blocked:        '#f43f5e',
}

function statusBorderColor(status: string): string {
  return STATUS_BORDER_COLORS[status] ?? '#94a3b8'
}

const PANEL_ID = 'backlog-panel-list'
</script>

<template>
  <section class="backlog-panel" aria-label="Backlog">
    <!-- Header bar -->
    <div class="backlog-header">
      <button
        class="backlog-toggle"
        :aria-expanded="!collapsed"
        :aria-controls="PANEL_ID"
        @click="toggleCollapsed"
      >
        <span class="toggle-chevron" :class="{ 'toggle-chevron--open': !collapsed }">▶</span>
        <span class="backlog-title">Backlog ({{ artifacts.length }})</span>
      </button>

      <!-- Filter dropdowns — visible only when expanded -->
      <template v-if="!collapsed">
        <select
          v-model="filterType"
          class="filter-select"
          aria-label="Filter by type"
        >
          <option value="">All types</option>
          <option v-for="t in typeOptions" :key="t" :value="t">{{ t }}</option>
        </select>

        <select
          v-model="filterStatus"
          class="filter-select"
          aria-label="Filter by status"
        >
          <option value="">All statuses</option>
          <option v-for="s in statusOptions" :key="s" :value="s">{{ s }}</option>
        </select>

        <select
          v-model="filterPriority"
          class="filter-select"
          aria-label="Filter by priority"
        >
          <option value="">All priorities</option>
          <option v-for="p in priorityOptions" :key="p" :value="p">{{ p }}</option>
        </select>
      </template>
    </div>

    <!-- Card list -->
    <div
      v-if="!collapsed"
      :id="PANEL_ID"
      class="backlog-list"
    >
      <!-- Empty state -->
      <p v-if="filtered.length === 0" class="backlog-empty">
        No backlog items match the current filters.
      </p>

      <button
        v-for="artifact in filtered"
        :key="artifact.path"
        class="backlog-card"
        :style="{ borderLeftColor: statusBorderColor(artifact.status) }"
        :aria-label="`Open artifact: ${artifact.title || artifact.slug}`"
        @click="emit('openArtifact', artifact.path)"
      >
        <div class="backlog-card-title">{{ artifact.title || artifact.slug }}</div>
        <div class="backlog-card-meta">
          <span
            class="backlog-type-badge"
            :style="{ background: typeBadgeColor(artifact.type) }"
          >{{ artifact.type }}</span>
          <span class="backlog-status-badge">{{ artifact.status }}</span>
          <span
            v-if="artifact.frontmatter?.priority"
            class="backlog-priority-badge"
          >{{ artifact.frontmatter.priority }}</span>
          <span class="backlog-lineage">{{ artifact.lineage }}</span>
        </div>
      </button>
    </div>
  </section>
</template>

<style scoped>
.backlog-panel {
  flex-shrink: 0;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  margin: var(--space-3) var(--space-4);
  background: var(--color-surface);
  overflow: hidden;
}

/* Header */
.backlog-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
  flex-wrap: wrap;
}
.backlog-header:has(+ .backlog-list:not([style*="display: none"])) {
  border-bottom: 1px solid var(--color-border);
}

.backlog-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
  font-family: inherit;
  color: var(--color-text);
  flex-shrink: 0;
}
.backlog-toggle:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
  border-radius: 2px;
}

.toggle-chevron {
  font-size: 9px;
  color: var(--color-text-muted);
  display: inline-block;
  transition: transform 0.15s;
  transform: rotate(0deg);
}
.toggle-chevron--open {
  transform: rotate(90deg);
}

.backlog-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}

/* Filter selects */
.filter-select {
  padding: 2px var(--space-2);
  font-size: 11px;
  font-family: inherit;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  cursor: pointer;
}
.filter-select:focus {
  outline: 2px solid var(--color-accent);
  outline-offset: 1px;
}

/* Card list — contain:content limits paint/layout to this subtree for large lists (NFR2) */
.backlog-list {
  max-height: 300px;
  overflow-y: auto;
  padding: var(--space-2);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  contain: content;
}

.backlog-empty {
  padding: var(--space-3);
  text-align: center;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  margin: 0;
}

/* Individual card */
.backlog-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-2) var(--space-3);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-left-width: 3px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-family: inherit;
  text-align: left;
  transition: border-color 0.12s, box-shadow 0.12s;
  width: 100%;
  box-sizing: border-box;
}
.backlog-card:hover {
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.10);
  background: var(--color-surface);
}
.backlog-card:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.backlog-card-title {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
  line-height: 1.4;
}

.backlog-card-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-1);
}

.backlog-type-badge {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  white-space: nowrap;
}

.backlog-status-badge {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 500;
  background: var(--color-border);
  color: var(--color-text);
  white-space: nowrap;
}

.backlog-priority-badge {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 500;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
  white-space: nowrap;
}

.backlog-lineage {
  font-size: 10px;
  color: var(--color-text-muted);
  font-family: monospace;
  margin-left: auto;
}
</style>
