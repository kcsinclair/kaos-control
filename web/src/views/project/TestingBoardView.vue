<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTestingStore } from '@/stores/testing'
import { useArtifactsStore } from '@/stores/artifacts'
import { useWebSocket } from '@/composables/useWebSocket'
import { parseArtifactDate } from '@/composables/useFormatDate'
import type { ArtifactRow, WsEvent } from '@/types/api'
import { FlaskConical } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const project = route.params.project as string

const store = useTestingStore()
const artifactsStore = useArtifactsStore()

function ageOf(created: string): string {
  const d = parseArtifactDate(created)
  if (!d) return '?'
  const days = Math.floor((Date.now() - d.getTime()) / 86_400_000)
  return `${days}d`
}

// ── filters ──────────────────────────────────────────────────────────────────
const filterStatus = ref('')
const filterLineage = ref('')
const filterLabel = ref('')
const filterPriority = ref('')

const filteredTests = computed(() => {
  return store.tests.filter((t) => {
    if (filterStatus.value && t.status !== filterStatus.value) return false
    if (filterLineage.value && t.lineage !== filterLineage.value) return false
    if (filterLabel.value) {
      const labels = t.frontmatter?.labels ?? []
      if (!labels.includes(filterLabel.value)) return false
    }
    if (filterPriority.value && (t.frontmatter?.priority ?? '') !== filterPriority.value) return false
    return true
  })
})

// Unique values for filter dropdowns derived from loaded tests
const allLineages = computed(() => [...new Set(store.tests.map((t) => t.lineage))].sort())
const allLabels = computed(() => {
  const s = new Set<string>()
  store.tests.forEach((t) => (t.frontmatter?.labels ?? []).forEach((l) => s.add(l)))
  return [...s].sort()
})
const allPriorities = computed(() => {
  const s = new Set<string>()
  store.tests.forEach((t) => { if (t.frontmatter?.priority) s.add(t.frontmatter.priority) })
  return [...s].sort()
})

const statusOptions = ['', 'draft', 'clarifying', 'planning', 'in-development', 'in-qa',
  'in-progress', 'done', 'approved', 'blocked', 'rejected', 'abandoned']

function resetFilters() {
  filterStatus.value = ''
  filterLineage.value = ''
  filterLabel.value = ''
  filterPriority.value = ''
}

// ── navigation ────────────────────────────────────────────────────────────────
function openArtifact(artifact: ArtifactRow) {
  router.push(`/p/${project}/artifacts/${artifact.path}`)
}

// ── WS refresh ────────────────────────────────────────────────────────────────
useWebSocket(project, 'artifact.indexed', (_e: WsEvent) => {
  store.fetchTests(project)
})

onMounted(async () => {
  await Promise.all([
    store.fetchTests(project),
    artifactsStore.fetchLabels(project),
    artifactsStore.fetchPriorities(project),
  ])
})
</script>

<template>
  <div class="testing-view">
    <!-- Header -->
    <div class="testing-header">
      <div class="testing-title-row">
        <FlaskConical :size="20" class="testing-title-icon" />
        <h2 class="testing-title">Testing</h2>
        <span class="testing-count">{{ filteredTests.length }} test{{ filteredTests.length !== 1 ? 's' : '' }}</span>
      </div>
    </div>

    <!-- Filter bar -->
    <div class="filter-bar">
      <select v-model="filterStatus">
        <option value="">All statuses</option>
        <option v-for="s in statusOptions.slice(1)" :key="s" :value="s">{{ s }}</option>
      </select>
      <select v-model="filterLineage" v-if="allLineages.length">
        <option value="">All lineages</option>
        <option v-for="l in allLineages" :key="l" :value="l">{{ l }}</option>
      </select>
      <select v-model="filterLabel" v-if="allLabels.length">
        <option value="">All labels</option>
        <option v-for="l in allLabels" :key="l" :value="l">{{ l }}</option>
      </select>
      <select v-model="filterPriority" v-if="allPriorities.length">
        <option value="">All priorities</option>
        <option v-for="p in allPriorities" :key="p" :value="p">{{ p }}</option>
      </select>
      <button class="btn-ghost" @click="resetFilters">Reset</button>
    </div>

    <!-- State messages -->
    <div v-if="store.loading" class="state-msg">Loading…</div>
    <div v-else-if="!store.loading && store.tests.length === 0" class="state-msg">
      No test artifacts found. Create artifacts with <code>type: test</code> to see them here.
    </div>

    <!-- Test grid -->
    <div v-else class="test-grid">
      <div
        v-for="test in filteredTests"
        :key="test.path"
        class="test-card"
        :class="{ 'test-card--dimmed': test.status !== 'approved' }"
        tabindex="0"
        role="button"
        :aria-label="`Open test: ${test.title || test.slug}`"
        @click="openArtifact(test)"
        @keydown.enter="openArtifact(test)"
      >
        <!-- Status badge -->
        <div class="card-header">
          <span class="status-badge" :class="`status-badge--${test.status}`">{{ test.status }}</span>
          <span
            v-if="test.frontmatter?.priority"
            class="priority-badge"
          >{{ test.frontmatter.priority }}</span>
        </div>

        <!-- Title -->
        <div class="card-title">{{ test.title || test.slug }}</div>

        <!-- Labels -->
        <div v-if="test.frontmatter?.labels?.length" class="card-labels">
          <span v-for="label in test.frontmatter.labels" :key="label" class="card-label">
            {{ label }}
          </span>
        </div>

        <!-- Footer: lineage + age -->
        <div class="card-footer">
          <span class="card-lineage">{{ test.lineage }}</span>
          <span class="card-age">{{ ageOf(test.created) }}</span>
        </div>
      </div>

      <div v-if="filteredTests.length === 0 && store.tests.length > 0" class="state-msg">
        No tests match the current filters.
      </div>
    </div>
  </div>
</template>

<style scoped>
.testing-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* Header */
.testing-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
  gap: var(--space-3);
  flex-wrap: wrap;
}
.testing-title-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.testing-title-icon {
  color: var(--color-text-muted);
}
.testing-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.testing-count {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}

/* Filter bar */
.filter-bar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-wrap: wrap;
  flex-shrink: 0;
}
.filter-bar select {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  cursor: pointer;
}
.btn-ghost {
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: var(--space-1) var(--space-3);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-ghost:hover { background: var(--color-surface); color: var(--color-text); }

.state-msg {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}

/* Test grid */
.test-grid {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-4) var(--space-6);
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--space-3);
  align-content: start;
}

.test-card {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  transition: border-color 0.12s, box-shadow 0.12s, opacity 0.12s;
  outline: none;
}
.test-card:hover {
  border-color: var(--color-accent);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
}
.test-card:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}
.test-card--dimmed {
  opacity: 0.6;
}

/* Card header: status + priority */
.card-header {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  flex-wrap: wrap;
}
.status-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  background: var(--color-border);
  color: var(--color-text-muted);
}
.status-badge--approved {
  background: #d1fae5;
  color: #065f46;
}
.status-badge--in-qa {
  background: #dbeafe;
  color: #1e40af;
}
.status-badge--blocked {
  background: #fef3c7;
  color: #92400e;
}
.status-badge--rejected,
.status-badge--abandoned {
  background: #fee2e2;
  color: #991b1b;
}
@media (prefers-color-scheme: dark) {
  .status-badge--approved { background: #064e3b; color: #6ee7b7; }
  .status-badge--in-qa { background: #1e3a8a; color: #93c5fd; }
  .status-badge--blocked { background: #451a03; color: #fcd34d; }
  .status-badge--rejected,
  .status-badge--abandoned { background: #450a0a; color: #fca5a5; }
}
.priority-badge {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 500;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
}

.card-title {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
  line-height: 1.4;
}

.card-labels {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}
.card-label {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 99px;
  font-size: 10px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
}
.card-lineage {
  font-size: 10px;
  color: var(--color-text-muted);
  font-family: monospace;
}
.card-age {
  font-size: 10px;
  color: var(--color-text-muted);
}
</style>
