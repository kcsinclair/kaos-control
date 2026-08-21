<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import * as artifactsApi from '@/api/artifacts'
import type { ArtifactRow } from '@/types/api'
import { Search } from 'lucide-vue-next'

// Features view: browse the project's shipped-capability records (type:
// feature). Default layout groups feature names by their `function:`
// frontmatter field; search + function/status filters narrow the set.
const route = useRoute()
const project = route.params.project as string

const features = ref<ArtifactRow[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

const search = ref('')
const functionFilter = ref('')
const statusFilter = ref('')

const UNGROUPED = 'Other'
function fnOf(f: ArtifactRow): string {
  return (f.frontmatter?.function || '').trim() || UNGROUPED
}

onMounted(async () => {
  try {
    const res = await artifactsApi.listArtifacts(project, { type: 'feature', limit: 500 })
    features.value = res.items ?? []
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load features.'
  } finally {
    loading.value = false
  }
})

const functions = computed(() => {
  const set = new Set<string>()
  features.value.forEach((f) => set.add(fnOf(f)))
  return Array.from(set).sort()
})
const statuses = computed(() => {
  const set = new Set<string>()
  features.value.forEach((f) => set.add(f.status))
  return Array.from(set).sort()
})

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return features.value.filter((f) => {
    if (functionFilter.value && fnOf(f) !== functionFilter.value) return false
    if (statusFilter.value && f.status !== statusFilter.value) return false
    if (q) {
      const hay = [f.title, f.frontmatter?.summary, fnOf(f), ...(f.frontmatter?.labels ?? [])]
        .join(' ')
        .toLowerCase()
      if (!hay.includes(q)) return false
    }
    return true
  })
})

// Default view: feature names grouped by function. Named groups sort
// alphabetically; the catch-all "Other" sinks to the bottom.
const grouped = computed(() => {
  const map = new Map<string, ArtifactRow[]>()
  for (const f of filtered.value) {
    const k = fnOf(f)
    if (!map.has(k)) map.set(k, [])
    map.get(k)!.push(f)
  }
  return Array.from(map.entries())
    .map(([name, items]) => ({
      name,
      items: items.slice().sort((a, b) => (a.title || '').localeCompare(b.title || '')),
    }))
    .sort((a, b) =>
      a.name === UNGROUPED ? 1 : b.name === UNGROUPED ? -1 : a.name.localeCompare(b.name),
    )
})

function resetFilters() {
  search.value = ''
  functionFilter.value = ''
  statusFilter.value = ''
}
</script>

<template>
  <div class="features-view">
    <div class="features-header">
      <h2 class="features-title">Features</h2>
      <span v-if="!loading" class="features-count">{{ filtered.length }} of {{ features.length }}</span>
    </div>

    <div class="filter-bar" v-if="!loading && features.length">
      <div class="search-wrap">
        <Search :size="15" class="search-icon" />
        <input v-model="search" class="search-input" type="text" placeholder="Search features…" />
      </div>
      <select v-model="functionFilter">
        <option value="">All functions</option>
        <option v-for="fn in functions" :key="fn" :value="fn">{{ fn }}</option>
      </select>
      <select v-model="statusFilter">
        <option value="">All statuses</option>
        <option v-for="s in statuses" :key="s" :value="s">{{ s }}</option>
      </select>
      <button v-if="search || functionFilter || statusFilter" class="btn-reset" @click="resetFilters">
        Clear
      </button>
    </div>

    <div v-if="loading" class="state" role="status">Loading features…</div>
    <div v-else-if="error" class="state error" role="alert">{{ error }}</div>
    <div v-else-if="!features.length" class="state">
      No features yet. Capture shipped capability as <code>type: feature</code> artifacts under
      <code>lifecycle/features/</code>.
    </div>
    <div v-else-if="!filtered.length" class="state">No features match the current filters.</div>

    <div v-else class="groups">
      <section v-for="group in grouped" :key="group.name" class="group">
        <h3 class="group-title">
          {{ group.name }}<span class="group-count">{{ group.items.length }}</span>
        </h3>
        <div class="cards">
          <RouterLink
            v-for="f in group.items"
            :key="f.path"
            class="card"
            :to="`/p/${project}/artifacts/${f.path}`"
          >
            <div class="card-head">
              <span class="card-title">{{ f.title || f.slug }}</span>
              <span class="status-pill" :data-status="f.status">{{ f.status }}</span>
            </div>
            <p v-if="f.frontmatter?.summary" class="card-summary">{{ f.frontmatter.summary }}</p>
          </RouterLink>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.features-view {
  padding: var(--space-6);
  max-width: 1100px;
}
.features-header {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}
.features-title {
  font-size: var(--text-xl);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}
.features-count {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.filter-bar {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  align-items: center;
  margin-bottom: var(--space-5);
}
.search-wrap {
  position: relative;
  display: flex;
  align-items: center;
}
.search-icon {
  position: absolute;
  left: 8px;
  color: var(--color-text-muted);
  pointer-events: none;
}
.search-input {
  padding: var(--space-1) var(--space-2) var(--space-1) 28px;
  min-width: 220px;
}
.filter-bar select,
.search-input {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  height: 32px;
}
.filter-bar select {
  padding: 0 var(--space-2);
}
.btn-reset {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: none;
  color: var(--color-text);
  font-size: var(--text-sm);
  height: 32px;
  padding: 0 var(--space-3);
  cursor: pointer;
}
.btn-reset:hover {
  background: var(--color-bg);
}
.state {
  padding: var(--space-8);
  text-align: center;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.state.error {
  color: var(--color-error);
}
.groups {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}
.group-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  margin: 0 0 var(--space-3);
  border-bottom: 1px solid var(--color-border);
  padding-bottom: var(--space-2);
}
.group-count {
  font-size: var(--text-xs);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  padding: 0 var(--space-2);
  color: var(--color-text-muted);
}
.cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--space-3);
}
.card {
  display: block;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  text-decoration: none;
  color: inherit;
  background: var(--color-surface, var(--color-bg));
  transition: border-color 0.12s, transform 0.12s;
}
.card:hover {
  border-color: var(--color-accent);
  transform: translateY(-1px);
}
.card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-2);
}
.card-title {
  font-weight: 600;
  color: var(--color-text);
  font-size: var(--text-sm);
}
.card-summary {
  margin: var(--space-2) 0 0;
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  line-height: 1.5;
}
.status-pill {
  flex: none;
  font-size: var(--text-xs);
  padding: 1px 8px;
  border-radius: 999px;
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
  white-space: nowrap;
}
.status-pill[data-status='approved'],
.status-pill[data-status='done'] {
  color: var(--color-success, #16a34a);
  border-color: var(--color-success, #16a34a);
}
</style>
