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

    <!-- No promoted architecture yet (FR-10): one overall empty state
         instead of six half-populated panels — it never errors. -->
    <div v-else-if="!hasChosenArchitecture" class="overview-empty">
      <p>No architecture has been chosen for this project yet.</p>
      <div class="overview-empty-actions">
        <router-link class="btn-primary" :to="`/p/${project}/architecture/wizard`">
          Run the Architecture Wizard
        </router-link>
        <router-link class="btn-ghost" :to="`/p/${project}/architecture/map`">
          Browse the relationship map
        </router-link>
      </div>
    </div>

    <!-- Second column reserved for a future auto-diagram (FR-11) — no visual
         chrome ships until that milestone; this stays a single-column grid
         until then so adding it later is additive, not a restructure. -->
    <div v-else class="overview-layout">
      <div class="overview-main">
        <ChosenArchitecturePanel :project="project" :item="chosenArchitecture" />
        <TechStackPanel :project="project" :stack="chosenStack" :architecture="chosenArchitecture" />
        <RationalePanel :project="project" :summary="summary" />
        <BreakingRequirementsPanel :project="project" :summary="summary" />
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
.overview-empty {
  padding: var(--space-8);
  text-align: center;
  color: var(--color-text-muted);
}
.overview-empty p {
  margin: 0 0 var(--space-4);
}
.overview-empty-actions {
  display: flex;
  justify-content: center;
  gap: var(--space-3);
}
.btn-primary,
.btn-ghost {
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
  text-decoration: none;
}
.btn-primary {
  background: var(--color-accent);
  color: #fff;
}
.btn-primary:hover {
  opacity: 0.88;
}
.btn-ghost {
  border: 1px solid var(--color-border);
  color: var(--color-text);
}
.btn-ghost:hover {
  background: var(--color-bg);
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
