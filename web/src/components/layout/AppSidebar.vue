<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import { api } from '@/api/client'
import { useWebSocket } from '@/composables/useWebSocket'
import type { WsEvent } from '@/types/api'
import {
  ChevronLeft,
  ChevronRight,
  List,
  Columns3,
  Network,
  Bot,
  AlertTriangle,
  Settings,
} from 'lucide-vue-next'
import type { Component } from 'vue'

const route = useRoute()
const projectStore = useProjectStore()
const uiStore = useUiStore()

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
  icon: Component
}

const navItems = (): NavItem[] => {
  const p = projectName()
  return [
    { label: 'List',         to: `/p/${p}/artifacts`,       icon: List },
    { label: 'Board',        to: `/p/${p}/artifacts/board`, icon: Columns3 },
    { label: 'Graph',        to: `/p/${p}/graph`,           icon: Network },
    { label: 'Agents',       to: `/p/${p}/agents`,          icon: Bot },
    { label: 'Parse Errors', to: `/p/${p}/parse-errors`,    icon: AlertTriangle },
    { label: 'Config',       to: `/p/${p}/config`,          icon: Settings },
  ]
}
</script>

<template>
  <nav
    class="app-sidebar"
    :class="{ 'sidebar--collapsed': uiStore.sidebarCollapsed }"
    aria-label="Project navigation"
  >
    <div class="sidebar-project">
      <span class="project-label">Project</span>
      <span class="project-name">{{ projectStore.current?.name ?? projectName() }}</span>
    </div>
    <ul class="nav-list" role="list">
      <li v-for="item in navItems()" :key="item.label" class="nav-item">
        <RouterLink
          :to="item.to"
          class="nav-link"
          :class="{ 'nav-link--active': route.path.startsWith(item.to) }"
          :aria-current="route.path.startsWith(item.to) ? 'page' : undefined"
        >
          <span class="nav-icon">
            <component :is="item.icon" :size="18" />
          </span>
          <span class="nav-label">{{ item.label }}</span>
          <span
            v-if="item.label === 'Parse Errors' && parseErrorCount > 0"
            class="badge"
            :aria-label="`${parseErrorCount} parse errors`"
          >{{ parseErrorCount }}</span>
        </RouterLink>
      </li>
    </ul>
    <div class="sidebar-footer">
      <button
        class="sidebar-toggle"
        :aria-label="uiStore.sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'"
        :aria-expanded="!uiStore.sidebarCollapsed"
        @click="uiStore.toggleSidebar()"
      >
        <ChevronRight v-if="uiStore.sidebarCollapsed" :size="16" />
        <ChevronLeft v-else :size="16" />
      </button>
    </div>
  </nav>
</template>

<style scoped>
.app-sidebar {
  width: var(--sidebar-width-expanded);
  flex-shrink: 0;
  background: var(--color-sidebar);
  border-right: 1px solid var(--color-border-dark);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  overflow-x: hidden;
}
.sidebar--collapsed {
  width: var(--sidebar-width-collapsed);
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
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-sm);
  color: var(--color-sidebar-text-muted);
  text-decoration: none;
  border-radius: var(--radius-md);
  transition: background 0.12s, color 0.12s;
  white-space: nowrap;
  overflow: hidden;
}
.nav-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 20px;
  height: 20px;
}
.nav-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
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
.sidebar-footer {
  padding: var(--space-2);
  border-top: 1px solid var(--color-border-dark);
  display: flex;
  justify-content: flex-end;
}
.sidebar--collapsed .sidebar-footer {
  justify-content: center;
}
.sidebar-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-sidebar-text-muted);
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}
.sidebar-toggle:hover {
  background: var(--color-sidebar-hover);
  color: var(--color-sidebar-text);
}
</style>
