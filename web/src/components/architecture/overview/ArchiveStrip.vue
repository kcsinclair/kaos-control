<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { OverviewItem } from '@/types/api'

// Archive strip (FR-9): a collapsed-by-default provenance strip for
// superseded promoted choices, showing up to 10 items once expanded (OQ-5).
// See [[architecture-overview-view]].
const props = defineProps<{ project: string; archive: OverviewItem[] }>()

const expanded = ref(false)
const MAX_SHOWN = 10

const shown = computed(() => props.archive.slice(0, MAX_SHOWN))
const remaining = computed(() => Math.max(0, props.archive.length - MAX_SHOWN))
</script>

<template>
  <section class="overview-panel archive-strip">
    <button
      type="button"
      class="archive-toggle"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      <span class="panel-title">Archive ({{ archive.length }})</span>
      <span class="archive-caret" :class="{ open: expanded }" aria-hidden="true">▸</span>
    </button>

    <template v-if="expanded">
      <p v-if="archive.length === 0" class="panel-empty">Nothing archived yet.</p>
      <template v-else>
        <ul class="item-list" role="list">
          <li v-for="item in shown" :key="item.path">
            <router-link :to="`/p/${project}/artifacts/${item.path}`" class="item-link">
              <span class="item-title">{{ item.title }}</span>
              <span class="status-chip" :data-status="item.status">{{ item.status }}</span>
            </router-link>
          </li>
        </ul>
        <p v-if="remaining > 0" class="archive-more">and {{ remaining }} more</p>
      </template>
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
.archive-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}
.panel-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.archive-caret {
  color: var(--color-text-muted);
  transition: transform 0.12s;
}
.archive-caret.open {
  transform: rotate(90deg);
}
.panel-empty {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  margin: var(--space-3) 0 0;
}
.item-list {
  list-style: none;
  margin: var(--space-3) 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.item-link {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-2);
  border-radius: var(--radius-md);
  text-decoration: none;
  color: var(--color-text);
  min-width: 0;
}
.item-link:hover {
  background: var(--color-bg);
}
.item-title {
  flex: 1;
  font-size: var(--text-sm);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}
.status-chip {
  flex-shrink: 0;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.03em;
  text-transform: uppercase;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--badge-raw-bg);
  color: var(--badge-raw-text);
}
.archive-more {
  margin: var(--space-2) 0 0;
  font-size: var(--text-xs);
  color: var(--color-text-muted);
}
</style>
