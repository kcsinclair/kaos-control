<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import type { OverviewItem } from '@/types/api'

// Standards panel (FR-6): list standards/*, each click-through to its
// artifact. See [[architecture-overview-view]].
defineProps<{ project: string; standards: OverviewItem[] }>()
</script>

<template>
  <section class="overview-panel">
    <header class="panel-header">
      <h3 class="panel-title">Standards</h3>
    </header>

    <p v-if="standards.length === 0" class="panel-empty">No standards recorded yet.</p>
    <ul v-else class="item-list" role="list">
      <li v-for="item in standards" :key="item.path">
        <router-link :to="`/p/${project}/artifacts/${item.path}`" class="item-link">
          <span class="item-title">{{ item.title }}</span>
          <span class="status-chip" :data-status="item.status">{{ item.status }}</span>
        </router-link>
      </li>
    </ul>
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
.panel-empty {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  margin: 0;
}
.item-list {
  list-style: none;
  margin: 0;
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
.status-chip[data-status="approved"]       { background: var(--badge-approved-bg);     color: var(--badge-approved-text); }
.status-chip[data-status="done"]           { background: var(--badge-done-bg);         color: var(--badge-done-text); }
.status-chip[data-status="blocked"]        { background: var(--badge-blocked-bg);      color: var(--badge-blocked-text); }
.status-chip[data-status="rejected"]       { background: var(--badge-rejected-bg);     color: var(--badge-rejected-text); }
.status-chip[data-status="in-development"] { background: var(--badge-in-dev-bg);       color: var(--badge-in-dev-text); }
.status-chip[data-status="in-qa"]          { background: var(--badge-in-qa-bg);        color: var(--badge-in-qa-text); }
</style>
