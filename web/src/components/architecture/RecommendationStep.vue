<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
// Milestone 5 (FR-9, FR-10, FR-11, NFR-4): transparent results — never a
// single black-box answer. Renders the top 2-3 candidates each with a
// visible "why", the OQ-2 dropped-constraints notice when the match isn't
// exact, and an override expander that falls back to Browse for any other
// catalog item (FR-4).
import { ref } from 'vue'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import type { CatalogItem } from '@/types/api'

const emit = defineEmits<{
  chosen: [item: CatalogItem]
  'browse-anyway': []
}>()

const store = useArchitectureWizardStore()
const overrideExpanded = ref(false)
</script>

<template>
  <div class="recommend-step">
    <h2 class="recommend-title">Here's what looks like the best fit</h2>

    <div
      v-if="store.droppedConstraints.length > 0"
      class="dropped-banner"
      role="status"
    >
      No exact match for every answer — here's the closest fit. Dropped to find a match:
      <ul>
        <li v-for="c in store.droppedConstraints" :key="c">{{ c }}</li>
      </ul>
    </div>

    <ul class="recommend-cards" role="list" aria-label="Recommended architectures">
      <li v-for="r in store.recommendations" :key="r.item.path" class="recommend-card">
        <button type="button" class="recommend-card-btn" @click="emit('chosen', r.item)">
          <span class="recommend-card-title">{{ r.item.title }}</span>
          <span class="recommend-card-summary">{{ r.item.summary }}</span>
          <ul class="recommend-why" aria-label="Why this fits">
            <li v-for="(w, i) in r.why" :key="i">{{ w }}</li>
          </ul>
          <span class="recommend-card-select">Choose this architecture</span>
        </button>
      </li>
    </ul>

    <div class="override-section">
      <button
        type="button"
        class="override-toggle"
        :aria-expanded="overrideExpanded"
        @click="overrideExpanded = !overrideExpanded"
      >
        None of these quite right? Override with any other candidate
      </button>
      <div v-if="overrideExpanded" class="override-body">
        <button type="button" class="show-everything-btn" @click="emit('browse-anyway')">
          Browse the full catalog
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.recommend-step {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.recommend-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
}
.dropped-banner {
  padding: var(--space-3) var(--space-4);
  background: #fef3c7;
  border: 1px solid #fcd34d;
  border-radius: var(--radius-md);
  color: #92400e;
  font-size: var(--text-sm);
  line-height: 1.6;
}
.dropped-banner ul {
  margin: var(--space-1) 0 0;
  padding-left: 1.2em;
}
.recommend-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--space-4);
  list-style: none;
  margin: 0;
  padding: 0;
}
.recommend-card-btn {
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
.recommend-card-btn:hover,
.recommend-card-btn:focus-visible {
  border-color: var(--color-accent);
}
.recommend-card-title {
  font-size: var(--text-base);
  font-weight: 700;
  color: var(--color-text);
}
.recommend-card-summary {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  line-height: 1.5;
}
.recommend-why {
  margin: 0;
  padding-left: 1.1em;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.recommend-card-select {
  margin-top: auto;
  padding-top: var(--space-2);
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-accent);
}
.override-section {
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-4);
}
.override-toggle {
  background: none;
  border: none;
  padding: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
  text-decoration: underline;
}
.override-body {
  margin-top: var(--space-3);
}
.show-everything-btn {
  background: none;
  border: none;
  padding: 0;
  color: var(--color-accent);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  text-decoration: underline;
}
</style>
