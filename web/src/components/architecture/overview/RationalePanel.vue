<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useArtifactsStore } from '@/stores/artifacts'
import MarkdownPreview from '@/components/artifact/MarkdownPreview.vue'
import { summaryWithoutSection, resolveSummaryLinks } from '@/lib/architectureSummary'
import type { ArtifactDetail, OverviewItem } from '@/types/api'

// Wizard Q&A rationale panel (FR-4): render the relevant
// architecture-summary.md sections as-is with links (OQ-1 — no structured
// heading parsing) — the whole summary body minus the breaking-requirements
// section, which BreakingRequirementsPanel owns. See [[architecture-overview-view]].
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

const rationale = computed(() => {
  if (!detail.value) return ''
  const withoutBreaking = summaryWithoutSection(detail.value.body, 'Architecture-breaking requirements')
  return resolveSummaryLinks(withoutBreaking, props.project)
})
</script>

<template>
  <section class="overview-panel">
    <header class="panel-header">
      <h3 class="panel-title">Wizard rationale</h3>
    </header>

    <p v-if="!summary" class="panel-empty">No architecture summary yet.</p>
    <template v-else>
      <div v-if="loading" class="panel-loading">Loading…</div>
      <p v-else-if="error" class="panel-error">{{ error }}</p>
      <MarkdownPreview v-else-if="detail" :source="rationale" :project="project" />
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
