<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useArtifactsStore } from '@/stores/artifacts'
import MarkdownPreview from '@/components/artifact/MarkdownPreview.vue'
import { extractSummarySection, resolveSummaryLinks } from '@/lib/architectureSummary'
import type { ArtifactDetail, OverviewItem } from '@/types/api'

// Architecture-breaking requirements panel (FR-5): render the summary's
// breaking-requirements section as-is, preserving click-through where the
// summary links to an ADR/requirement. See [[architecture-overview-view]].
const props = defineProps<{ project: string; summary: OverviewItem | null }>()

const store = useArtifactsStore()
const detail = ref<ArtifactDetail | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

watch(
  () => props.summary?.path,
  async (path) => {
    detail.value = null
    error.value = null
    if (!path) return
    loading.value = true
    try {
      detail.value = await store.fetchOne(props.project, path)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load the summary.'
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

const breaking = computed(() => {
  if (!detail.value) return ''
  // Falls back to the whole body if the heading is missing (NFR-5) — the
  // summary still degrades to something rather than an empty panel.
  const section = extractSummarySection(detail.value.body, 'Architecture-breaking requirements') ?? detail.value.body
  return resolveSummaryLinks(section, props.project)
})
</script>

<template>
  <section class="overview-panel">
    <header class="panel-header">
      <h3 class="panel-title">Architecture-breaking requirements</h3>
    </header>

    <p v-if="!summary" class="panel-empty">No architecture summary yet.</p>
    <template v-else>
      <div v-if="loading" class="panel-loading">Loading…</div>
      <p v-else-if="error" class="panel-error">{{ error }}</p>
      <MarkdownPreview v-else-if="detail" :source="breaking" :project="project" />
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
