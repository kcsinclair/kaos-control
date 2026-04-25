<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { api } from '@/api/client'
import { useWebSocket } from '@/composables/useWebSocket'
import type { WsEvent } from '@/types/api'

const route = useRoute()
const projectStore = useProjectStore()

const projectName = () => route.params.project as string
const parseErrorCount = ref(0)

async function fetchParseErrors(project: string) {
  try {
    const res = await api.get<{ errors: Array<unknown> | null }>(
      `/p/${encodeURIComponent(project)}/parse-errors`
    )
    parseErrorCount.value = res.errors?.length ?? 0
  } catch {
    parseErrorCount.value = 0
  }
}

onMounted(() => fetchParseErrors(projectName()))

watch(() => route.params.project, (p) => {
  if (p) fetchParseErrors(p as string)
})

// Re-check when artifacts are re-indexed (a re-index may fix or introduce errors)
useWebSocket(projectName(), 'artifact.indexed', (_e: WsEvent) => {
  fetchParseErrors(projectName())
})

interface NavItem {
  label: string
  to: string
}

const navItems = (): NavItem[] => {
  const p = projectName()
  return [
    { label: 'Artifacts', to: `/p/${p}/artifacts` },
    { label: 'Graph', to: `/p/${p}/graph` },
    { label: 'Agents', to: `/p/${p}/agents` },
    { label: 'Parse Errors', to: `/p/${p}/parse-errors` },
    { label: 'Config', to: `/p/${p}/config` },
  ]
}
</script>

<template>
  <nav class="app-sidebar" aria-label="Project navigation">
    <div class="sidebar-project">
      <span class="project-label">Project</span>
      <span class="project-name">{{ projectStore.current?.name ?? projectName() }}</span>
    </div>
    <ul class="nav-list" role="list">
      <li v-for="item in navItems()" :key="item.to" class="nav-item">
        <RouterLink
          :to="item.to"
          class="nav-link"
          :class="{ 'nav-link--active': route.path.startsWith(item.to) }"
          :aria-current="route.path.startsWith(item.to) ? 'page' : undefined"
        >
          {{ item.label }}
          <span
            v-if="item.label === 'Parse Errors' && parseErrorCount > 0"
            class="badge"
            :aria-label="`${parseErrorCount} parse errors`"
          >{{ parseErrorCount }}</span>
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
  display: flex;
  align-items: center;
  justify-content: space-between;
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
.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: var(--radius-full);
  background: var(--color-error);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  line-height: 1;
  flex-shrink: 0;
}
.nav-link--active .badge {
  background: rgba(255,255,255,0.25);
}
</style>
