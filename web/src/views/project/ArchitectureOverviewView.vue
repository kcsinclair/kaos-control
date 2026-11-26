<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useArchitectureOverview } from '@/composables/useArchitectureOverview'
import { useUiStore } from '@/stores/ui'
import ChosenArchitecturePanel from '@/components/architecture/overview/ChosenArchitecturePanel.vue'
import TechStackPanel from '@/components/architecture/overview/TechStackPanel.vue'
import RationalePanel from '@/components/architecture/overview/RationalePanel.vue'
import BreakingRequirementsPanel from '@/components/architecture/overview/BreakingRequirementsPanel.vue'
import StandardsPanel from '@/components/architecture/overview/StandardsPanel.vue'
import AdrListPanel from '@/components/architecture/overview/AdrListPanel.vue'
import ArchiveStrip from '@/components/architecture/overview/ArchiveStrip.vue'
import NewAdrModal from '@/components/artifact/NewAdrModal.vue'

const route = useRoute()
const router = useRouter()
const project = route.params.project as string
const ui = useUiStore()

const {
  loading,
  error,
  hasChosenArchitecture,
  chosenArchitecture,
  chosenStack,
  summary,
  standards,
  adrs,
  archive,
  reload,
} = useArchitectureOverview(project)

// Render the panels whenever the project has ANY authored architecture content
// — not only a promoted root architecture. A project can have a summary, ADRs,
// or standards without (yet) a chosen architecture; each panel degrades
// independently, so gating the whole view on hasChosenArchitecture would hide
// the summary/ADRs/standards that DO exist. Catalog candidates alone don't
// count — a project that's only browsing options belongs on the map.
const hasAnyContent = computed(
  () =>
    hasChosenArchitecture.value ||
    !!chosenStack.value ||
    !!summary.value ||
    standards.value.length > 0 ||
    adrs.value.length > 0 ||
    archive.value.length > 0,
)

// One-click actions (FR-8): the map and wizard are each reachable in one
// click; raising an ADR reuses the existing NewAdrModal.vue rather than
// introducing a new authoring surface (NFR-2).
const showNewAdrModal = ref(false)

function onAdrCreated(path: string) {
  showNewAdrModal.value = false
  ui.success('ADR created!')
  void reload()
  void router.push(`/p/${project}/artifacts/${path}`)
}
</script>

<template>
  <div class="overview-view">
    <div class="overview-header">
      <h2 class="overview-title">Architecture Overview</h2>
      <div class="overview-actions">
        <router-link class="btn-ghost" :to="`/p/${project}/architecture/map`">Relationship map</router-link>
        <router-link class="btn-ghost" :to="`/p/${project}/architecture/wizard`">Architecture Wizard</router-link>
        <button class="btn-primary" type="button" @click="showNewAdrModal = true">New ADR</button>
      </div>
    </div>

    <div v-if="loading" class="overview-state" role="status" aria-live="polite">Loading architecture overview…</div>
    <div v-else-if="error" class="overview-state error" role="alert">{{ error }}</div>

    <!-- Nothing authored in the architecture zone yet (FR-10): one overall
         empty state. Once ANY content exists (summary/ADRs/standards/chosen),
         the panels render and each degrades independently. -->
    <div v-else-if="!hasAnyContent" class="overview-empty">
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
        <ArchiveStrip :project="project" :archive="archive" />
      </div>
    </div>

    <NewAdrModal
      v-if="showNewAdrModal"
      :project="project"
      @close="showNewAdrModal = false"
      @created="onAdrCreated"
    />
  </div>
</template>

<style scoped>
.overview-view {
  padding: var(--space-6);
  max-width: 1100px;
}
.overview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}
.overview-title {
  font-size: var(--text-xl);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}
.overview-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.overview-actions .btn-ghost,
.overview-actions .btn-primary {
  padding: var(--space-1) var(--space-3);
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
  border: none;
  cursor: pointer;
}
.btn-primary:hover {
  opacity: 0.88;
}
.btn-ghost {
  border: 1px solid var(--color-border);
  color: var(--color-text);
  background: none;
  cursor: pointer;
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
