<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useArtifactsStore } from '@/stores/artifacts'
import MarkdownPreview from '@/components/artifact/MarkdownPreview.vue'
import type { ArtifactDetail, OverviewItem } from '@/types/api'

// Chosen architecture panel (FR-2): title, summary, rendered body
// (components/interactions) via MarkdownPreview, click-through to the
// artifact. See [[architecture-overview-view]].
const props = defineProps<{ project: string; item: OverviewItem | null }>()

const store = useArtifactsStore()
const detail = ref<ArtifactDetail | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

watch(
  () => props.item?.path,
  async (path) => {
    detail.value = null
    error.value = null
    if (!path) return
    loading.value = true
    try {
      detail.value = await store.fetchOne(props.project, path)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load the chosen architecture.'
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)
</script>

<template>
  <section class="overview-panel">
    <header class="panel-header">
      <h3 class="panel-title">Chosen architecture</h3>
    </header>

    <p v-if="!item" class="panel-empty">No architecture has been chosen yet.</p>
    <template v-else>
      <router-link :to="`/p/${project}/artifacts/${item.path}`" class="panel-link-title">
        {{ item.title }}
      </router-link>
      <p v-if="detail?.frontmatter?.summary" class="panel-summary">{{ detail.frontmatter.summary }}</p>

      <div v-if="loading" class="panel-loading">Loading…</div>
      <p v-else-if="error" class="panel-error">{{ error }}</p>
      <MarkdownPreview v-else-if="detail" :source="detail.body" :project="project" />
    </template>
  </section>
</template>

<style scoped>
.overview-panel {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
}
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-3);
}
.panel-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.panel-link-title {
  display: block;
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--color-text);
  text-decoration: none;
  margin-bottom: var(--space-2);
}
.panel-link-title:hover {
  color: var(--color-accent);
  text-decoration: underline;
}
.panel-summary {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  margin: 0 0 var(--space-3);
}
.panel-loading,
.panel-error {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.panel-error {
  color: var(--color-error);
}
.panel-empty {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  margin: 0;
}
</style>
