<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getOverview } from '@/api/architecture'

// Architecture section landing (FR-1): resolves to the overview when the
// project has a chosen architecture, else the relationship map — a
// lightweight redirecting view rather than a blocking router guard, so the
// decision (a cheap has_chosen_architecture read) gets its own loading
// feedback instead of stalling navigation. See [[architecture-overview-view]].
const route = useRoute()
const router = useRouter()
const project = route.params.project as string

onMounted(async () => {
  let target = 'architecture/map'
  try {
    const ov = await getOverview(project)
    // Prefer the overview whenever the project has ANY authored architecture
    // content — a chosen architecture/stack, a summary, ADRs, standards, or
    // archived choices. Catalog candidates alone don't count: a project that's
    // only browsing options still belongs on the map. Mirrors the
    // hasAnyContent gate in ArchitectureOverviewView.
    const hasContent =
      ov.has_chosen_architecture ||
      !!ov.chosen_stack ||
      !!ov.summary ||
      (ov.standards?.length ?? 0) > 0 ||
      (ov.adrs?.length ?? 0) > 0 ||
      (ov.archive?.length ?? 0) > 0
    if (hasContent) target = 'architecture/overview'
  } catch {
    // Degrades to the map on error (NFR-5) — it carries its own error state.
  }
  router.replace(`/p/${project}/${target}`)
})
</script>

<template>
  <div class="architecture-landing-state" role="status" aria-live="polite">Loading architecture…</div>
</template>

<style scoped>
.architecture-landing-state {
  padding: var(--space-8);
  text-align: center;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
</style>
