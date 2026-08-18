<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
// Milestone 3 (FR-5, FR-6): the Browse experience — every catalog
// architecture up front (cards + comparison table), fed by the
// wizard/catalog endpoint (OQ-6). No client-side markdown parsing; pros/cons
// come straight off CatalogItem.
import { computed, onMounted, ref } from 'vue'
import { listCatalog } from '@/api/architecture'
import { ApiError } from '@/api/client'
import type { CatalogItem } from '@/types/api'

const props = defineProps<{
  project: string
}>()

const emit = defineEmits<{
  chosen: [item: CatalogItem]
}>()

const architectures = ref<CatalogItem[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

const allLabels = computed(() => {
  const seen = new Set<string>()
  for (const a of architectures.value) {
    for (const l of a.labels) seen.add(l)
  }
  return Array.from(seen).sort()
})

async function load(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const catalog = await listCatalog(props.project)
    architectures.value = catalog.architectures
  } catch (e: unknown) {
    error.value = e instanceof ApiError ? e.message : 'Failed to load the architecture catalog.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="browse-step">
    <div class="browse-header">
      <h2 class="browse-title">Browse the architecture catalog</h2>
      <router-link :to="`/p/${project}/architecture/map`" class="map-link">
        View relationship map
      </router-link>
    </div>

    <div v-if="loading" class="browse-state" role="status" aria-live="polite">
      Loading catalog…
    </div>
    <div v-else-if="error" class="browse-state error" role="alert">{{ error }}</div>
    <div v-else-if="architectures.length === 0" class="browse-state empty">
      No catalog present in this project yet. Run project initialisation to copy the
      architecture catalog in, or see the open defect
      <code>project-init-missing-architecture-artefacts.md</code>.
    </div>

    <template v-else>
      <ul class="arch-cards" role="list" aria-label="Candidate architectures">
        <li v-for="a in architectures" :key="a.path" class="arch-card">
          <button type="button" class="arch-card-btn" @click="emit('chosen', a)">
            <span class="arch-card-title">{{ a.title }}</span>
            <span class="arch-card-summary">{{ a.summary }}</span>
            <span v-if="a.labels.length" class="arch-card-labels">
              <span v-for="l in a.labels" :key="l" class="label-chip">{{ l }}</span>
            </span>
            <div class="arch-card-proscons">
              <div v-if="a.pros?.length" class="proscons-col">
                <span class="proscons-heading">Pros</span>
                <ul>
                  <li v-for="(p, i) in a.pros" :key="i">{{ p }}</li>
                </ul>
              </div>
              <div v-if="a.cons?.length" class="proscons-col">
                <span class="proscons-heading">Cons</span>
                <ul>
                  <li v-for="(c, i) in a.cons" :key="i">{{ c }}</li>
                </ul>
              </div>
            </div>
            <span class="arch-card-select">Choose this architecture</span>
          </button>
        </li>
      </ul>

      <div class="compare-table-scroll">
        <table class="compare-table" aria-label="Architecture comparison">
          <thead>
            <tr>
              <th scope="col">Architecture</th>
              <th scope="col">Summary</th>
              <th v-for="l in allLabels" :key="l" scope="col">{{ l }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in architectures" :key="a.path">
              <th scope="row">{{ a.title }}</th>
              <td>{{ a.summary }}</td>
              <td v-for="l in allLabels" :key="l" class="compare-cell">
                <span v-if="a.labels.includes(l)" aria-label="yes">✓</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>

<style scoped>
.browse-step {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.browse-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}
.browse-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
}
.map-link {
  font-size: var(--text-sm);
  color: var(--color-accent);
}
.browse-state {
  padding: var(--space-4);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.browse-state.error { color: #991b1b; }
.browse-state.empty {
  background: var(--color-surface);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-md);
  line-height: 1.6;
}
.arch-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--space-4);
  list-style: none;
  margin: 0;
  padding: 0;
}
.arch-card-btn {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  width: 100%;
  height: 100%;
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  text-align: left;
  cursor: pointer;
}
.arch-card-btn:hover,
.arch-card-btn:focus-visible {
  border-color: var(--color-accent);
}
.arch-card-title {
  font-size: var(--text-base);
  font-weight: 700;
  color: var(--color-text);
}
.arch-card-summary {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  line-height: 1.5;
}
.arch-card-labels {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}
.label-chip {
  padding: 1px 8px;
  background: var(--color-border);
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text);
}
.arch-card-proscons {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
  font-size: var(--text-sm);
}
.proscons-heading {
  display: block;
  font-weight: 600;
  color: var(--color-text);
  margin-bottom: var(--space-1);
}
.arch-card-proscons ul {
  margin: 0;
  padding-left: 1.1em;
  color: var(--color-text-muted);
}
.arch-card-select {
  margin-top: auto;
  padding-top: var(--space-2);
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-accent);
}
.compare-table-scroll {
  overflow-x: auto;
}
.compare-table {
  width: 100%;
  min-width: 640px;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
.compare-table th,
.compare-table td {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  text-align: left;
}
.compare-table th[scope='row'] {
  font-weight: 600;
  color: var(--color-text);
}
.compare-cell {
  text-align: center;
  color: var(--color-accent);
  font-weight: 700;
}
</style>
