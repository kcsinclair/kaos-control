<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/project'

const route = useRoute()
const projectStore = useProjectStore()

const projectName = () => route.params.project as string

interface NavItem {
  label: string
  to: string
  milestone: string
}

const navItems = (): NavItem[] => {
  const p = projectName()
  return [
    { label: 'Artifacts', to: `/p/${p}/artifacts`, milestone: 'M2' },
    { label: 'Graph', to: `/p/${p}/graph`, milestone: 'M3' },
    { label: 'Agents', to: `/p/${p}/agents`, milestone: 'M5' },
    { label: 'Parse Errors', to: `/p/${p}/parse-errors`, milestone: 'M6' },
    { label: 'Config', to: `/p/${p}/config`, milestone: 'M6' },
  ]
}
</script>

<template>
  <nav class="app-sidebar">
    <div class="sidebar-project">
      <span class="project-label">Project</span>
      <span class="project-name">{{ projectStore.current?.name ?? projectName() }}</span>
    </div>
    <ul class="nav-list">
      <li v-for="item in navItems()" :key="item.to" class="nav-item">
        <RouterLink
          :to="item.to"
          class="nav-link"
          :class="{ 'nav-link--active': route.path.startsWith(item.to) }"
        >
          {{ item.label }}
        </RouterLink>
      </li>
    </ul>
  </nav>
</template>

<style scoped>
.app-sidebar {
  width: 220px;
  flex-shrink: 0;
  background: var(--color-sidebar);
  border-right: 1px solid var(--color-border-dark);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}
.sidebar-project {
  padding: var(--space-4) var(--space-4) var(--space-2);
  border-bottom: 1px solid var(--color-border-dark);
}
.project-label {
  display: block;
  font-size: var(--text-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-sidebar-text-muted);
  margin-bottom: var(--space-1);
}
.project-name {
  display: block;
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-sidebar-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.nav-list {
  list-style: none;
  margin: var(--space-2) 0 0;
  padding: 0 var(--space-2);
  flex: 1;
}
.nav-item {
  margin-bottom: 2px;
}
.nav-link {
  display: block;
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-sm);
  color: var(--color-sidebar-text-muted);
  text-decoration: none;
  border-radius: var(--radius-md);
  transition: background 0.12s, color 0.12s;
}
.nav-link:hover {
  background: var(--color-sidebar-hover);
  color: var(--color-sidebar-text);
}
.nav-link--active {
  background: var(--color-sidebar-active);
  color: #fff;
}
</style>
