<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useArchitectureOverview } from '@/composables/useArchitectureOverview'
import ChosenArchitecturePanel from '@/components/architecture/overview/ChosenArchitecturePanel.vue'
import TechStackPanel from '@/components/architecture/overview/TechStackPanel.vue'
import RationalePanel from '@/components/architecture/overview/RationalePanel.vue'
import BreakingRequirementsPanel from '@/components/architecture/overview/BreakingRequirementsPanel.vue'
import StandardsPanel from '@/components/architecture/overview/StandardsPanel.vue'
import AdrListPanel from '@/components/architecture/overview/AdrListPanel.vue'

const route = useRoute()
const project = route.params.project as string

const {
  loading,
  error,
  hasChosenArchitecture,
  chosenArchitecture,
  chosenStack,
  summary,
  standards,
  adrs,
} = useArchitectureOverview(project)
</script>

<template>
  <div class="overview-view">
    <div class="overview-header">
      <h2 class="overview-title">Architecture Overview</h2>
    </div>

    <div v-if="loading" class="overview-state" role="status" aria-live="polite">Loading architecture overview…</div>
    <div v-else-if="error" class="overview-state error" role="alert">{{ error }}</div>

    <!-- Second column reserved for a future auto-diagram (FR-11) — no visual
         chrome ships until that milestone; this stays a single-column grid
         until then so adding it later is additive, not a restructure. -->
    <div v-else class="overview-layout">
      <div class="overview-main">
        <ChosenArchitecturePanel :project="project" :item="chosenArchitecture" />
        <TechStackPanel :project="project" :stack="chosenStack" :architecture="chosenArchitecture" />
        <RationalePanel v-if="hasChosenArchitecture" :project="project" :summary="summary" />
        <BreakingRequirementsPanel v-if="hasChosenArchitecture" :project="project" :summary="summary" />
        <StandardsPanel :project="project" :standards="standards" />
        <AdrListPanel :project="project" :adrs="adrs" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.overview-view {
  padding: var(--space-6);
  max-width: 1100px;
}
.overview-header {
  margin-bottom: var(--space-4);
}
.overview-title {
  font-size: var(--text-xl);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}
.overview-state {
  padding: var(--space-8);
  text-align: center;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.overview-state.error {
  color: var(--color-error);
}
.overview-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: var(--space-6);
}
.overview-main {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}
</style>
