<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDocsStore } from '@/stores/docs'
import { useWebSocket } from '@/composables/useWebSocket'
import type { WsEvent } from '@/types/api'
import type { DocEntry } from '@/api/docs'

const route = useRoute()
const router = useRouter()
const docsStore = useDocsStore()

const project = route.params.project as string

// Local search query with 100 ms debounce into the store
const localQuery = ref('')
let debounceTimer: ReturnType<typeof setTimeout> | null = null

watch(localQuery, (val) => {
  if (debounceTimer !== null) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    docsStore.setQuery(val)
    debounceTimer = null
  }, 100)
})

onMounted(() => {
  docsStore.fetch(project)
})

useWebSocket(project, 'doc.changed', (_e: WsEvent) => {
  docsStore.applyDocChanged(project)
})

const groupedDocs = computed(() => docsStore.groupedDocs)
const totalDocs = computed(() => docsStore.docs.length)
const filteredTotal = computed(() => docsStore.filteredDocs.length)

function openDoc(doc: DocEntry): void {
  router.push({
    name: 'docs-editor',
    params: { project, pathMatch: doc.path.split('/') },
  })
}

function clearQuery(): void {
  localQuery.value = ''
  docsStore.clearQuery()
}
</script>

<template>
  <div class="docs-view">
    <div class="docs-header">
      <h1 class="docs-title">Documentation</h1>
      <div class="docs-search">
        <input
          v-model="localQuery"
          type="search"
          class="search-input"
          aria-label="Search documents"
          placeholder="Search documents…"
        />
      </div>
    </div>

    <!-- Accessible live status for screen readers -->
    <p
      class="docs-status"
      aria-live="polite"
      aria-atomic="true"
    >
      {{ filteredTotal }} of {{ totalDocs }} documents
    </p>

    <!-- Empty state: no docs/ folder -->
    <div v-if="!docsStore.loading && !docsStore.docsDirPresent" class="empty-state">
      <p>No <code>docs/</code> folder in this project.</p>
    </div>

    <!-- Empty state: docs/ folder exists but no files -->
    <div
      v-else-if="!docsStore.loading && docsStore.docsDirPresent && groupedDocs.length === 0 && !localQuery"
      class="empty-state"
    >
      <p>This project has a <code>docs/</code> folder but it contains no markdown or supported files yet.</p>
    </div>

    <!-- Empty state: search returned nothing -->
    <div
      v-else-if="!docsStore.loading && groupedDocs.length === 0 && localQuery"
      class="empty-state"
    >
      <p>No documents match "<strong>{{ localQuery }}</strong>".</p>
      <button class="btn-clear" @click="clearQuery">Clear search</button>
    </div>

    <!-- Doc groups -->
    <div v-else class="docs-groups">
      <div v-for="group in groupedDocs" :key="group.subDir" class="docs-group">
        <h2 v-if="group.subDir" class="docs-subgroup">{{ group.subDir }}</h2>
        <div class="docs-cards">
          <template v-for="doc in group.docs" :key="doc.path">
            <!-- All docs open in the in-app viewer, which renders markdown as a
                 preview, HTML in a sandboxed iframe, and images inline. -->
            <button
              class="doc-card"
              type="button"
              @click="openDoc(doc)"
              @keydown.enter.prevent="openDoc(doc)"
              @keydown.space.prevent="openDoc(doc)"
            >
              <h3 class="doc-title">{{ doc.title }}</h3>
              <p class="doc-summary">{{ doc.summary }}</p>
              <span class="doc-path">{{ doc.path }}</span>
            </button>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.docs-view {
  padding: var(--space-6);
  max-width: 1000px;
}

.docs-header {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  margin-bottom: var(--space-2);
  flex-wrap: wrap;
}

.docs-title {
  font-size: var(--text-xl);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.docs-search {
  flex: 1;
  min-width: 200px;
  max-width: 360px;
}

.search-input {
  width: 100%;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-input);
  color: var(--color-text);
  font-size: var(--text-sm);
}

.search-input:focus {
  outline: 2px solid var(--color-accent);
  outline-offset: 1px;
}

.docs-status {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  margin: 0 0 var(--space-4);
}

.empty-state {
  padding: var(--space-8);
  text-align: center;
  color: var(--color-text-muted);
}

.empty-state p {
  margin: 0 0 var(--space-3);
}

.btn-clear {
  padding: var(--space-2) var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: none;
  color: var(--color-text);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: background 0.1s;
}

.btn-clear:hover {
  background: var(--color-border);
}

.docs-groups {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.docs-subgroup {
  font-size: var(--text-sm);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
  margin: 0 0 var(--space-3);
}

.docs-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--space-3);
}

.doc-card {
  display: block;
  text-align: left;
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  cursor: pointer;
  text-decoration: none;
  color: inherit;
  transition: border-color 0.12s, box-shadow 0.12s;
}

.doc-card:hover,
.doc-card:focus-visible {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px var(--color-accent-faint);
  outline: none;
}

.doc-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 var(--space-1);
  word-break: break-word;
}

.doc-summary {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  margin: 0 0 var(--space-2);
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.doc-path {
  font-size: 11px;
  font-family: var(--font-mono, monospace);
  color: var(--color-text-muted);
  opacity: 0.7;
}
</style>
