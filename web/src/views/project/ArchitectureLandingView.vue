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
    const overview = await getOverview(project)
    if (overview.has_chosen_architecture) target = 'architecture/overview'
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
