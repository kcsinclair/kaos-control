<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
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
  Activity,
  AlertTriangle,
  Settings,
  Layers,
  CalendarClock,
} from 'lucide-vue-next'
import type { Component } from 'vue'
import SidebarTooltip from '@/components/ui/SidebarTooltip.vue'

const route = useRoute()
const projectStore = useProjectStore()
const uiStore = useUiStore()
const authStore = useAuthStore()

const faviconSrc = `${import.meta.env.BASE_URL}favicon-32x32.png`

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
  roles?: string[]
}

const navItems = computed((): NavItem[] => {
  const p = projectName()
  const roles = authStore.rolesForProject(p)
  const hasDevOpsAccess = roles.includes('product-owner') || roles.includes('devops')
  const items: NavItem[] = [
    { label: 'List',         to: `/p/${p}/artifacts`,       icon: List },
    { label: 'Board',        to: `/p/${p}/artifacts/board`, icon: Columns3 },
    { label: 'Graph',        to: `/p/${p}/graph`,           icon: Network },
    { label: 'Agents',       to: `/p/${p}/agents`,          icon: Bot },
    { label: 'Scheduler',    to: `/p/${p}/scheduler`,       icon: CalendarClock },
    { label: 'Feed',         to: `/p/${p}/feed`,            icon: Activity },
    { label: 'Parse Errors', to: `/p/${p}/parse-errors`,    icon: AlertTriangle },
    { label: 'Config',       to: `/p/${p}/config`,          icon: Settings },
  ]
  if (hasDevOpsAccess) {
    items.push({ label: 'DevOps', to: `/p/${p}/devops`, icon: Layers })
  }
  return items
})

// Hover-to-expand overlay state (does NOT change persisted sidebarCollapsed)
const hoverExpanded = ref(false)
let hoverTimer: ReturnType<typeof setTimeout> | null = null

function onSidebarMouseEnter() {
  if (!uiStore.sidebarCollapsed) return
  hoverTimer = setTimeout(() => {
    hoverExpanded.value = true
  }, 200)
}

function onSidebarMouseLeave() {
  if (hoverTimer !== null) {
    clearTimeout(hoverTimer)
    hoverTimer = null
  }
  hoverExpanded.value = false
}

// Whether the sidebar visually appears expanded (either persisted or hover)
const isVisuallyExpanded = () => !uiStore.sidebarCollapsed || hoverExpanded.value

// Track transition direction for sequenced animation delays
const sidebarTransitionDir = ref<'expanding' | 'collapsing' | 'none'>('none')

watch(
  () => isVisuallyExpanded(),
  (expanded) => {
    sidebarTransitionDir.value = expanded ? 'expanding' : 'collapsing'
  }
)
</script>

<template>
  <nav
    class="app-sidebar"
    :class="{
      'sidebar--collapsed': uiStore.sidebarCollapsed && !hoverExpanded,
      'sidebar--overlay': uiStore.sidebarCollapsed && hoverExpanded,
      'sidebar--expanding': sidebarTransitionDir === 'expanding',
      'sidebar--collapsing': sidebarTransitionDir === 'collapsing',
    }"
    aria-label="Project navigation"
    @mouseenter="onSidebarMouseEnter"
    @mouseleave="onSidebarMouseLeave"
  >
    <div class="sidebar-project">
      <img
        v-if="!isVisuallyExpanded()"
        :src="faviconSrc"
        alt="Project"
        class="sidebar-favicon"
      />
      <template v-else>
        <span class="project-label">Project</span>
        <span class="project-name">{{ projectStore.current?.name ?? projectName() }}</span>
      </template>
    </div>
    <ul class="nav-list" role="list">
      <li v-for="item in navItems" :key="item.label" class="nav-item">
        <SidebarTooltip :label="item.label" :disabled="isVisuallyExpanded()">
          <RouterLink
            :to="item.to"
            class="nav-link"
            :class="{ 'nav-link--active': route.path.startsWith(item.to) }"
            :aria-current="route.path.startsWith(item.to) ? 'page' : undefined"
            :aria-label="!isVisuallyExpanded() ? item.label : undefined"
          >
            <span class="nav-icon tooltip-anchor">
              <component :is="item.icon" :size="18" />
              <span
                v-if="item.label === 'Parse Errors' && parseErrorCount > 0 && !isVisuallyExpanded()"
                class="badge-dot"
                :aria-label="`${parseErrorCount} parse errors`"
              >{{ parseErrorCount > 9 ? parseErrorCount : '' }}</span>
            </span>
            <span class="nav-label">{{ item.label }}</span>
            <span
              v-if="item.label === 'Parse Errors' && parseErrorCount > 0 && isVisuallyExpanded()"
              class="badge"
              :aria-label="`${parseErrorCount} parse errors`"
            >{{ parseErrorCount }}</span>
          </RouterLink>
        </SidebarTooltip>
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
  transition: width 250ms ease;
}
/* Collapse: text fades first (0ms delay), then width shrinks (100ms delay) */
.sidebar--collapsing {
  transition: width 250ms ease 100ms;
}
.sidebar--collapsing .nav-label,
.sidebar--collapsing .project-label,
.sidebar--collapsing .project-name {
  transition: opacity 100ms ease 0ms;
  opacity: 0;
}
/* Expand: width grows first (0ms delay), then text fades in (250ms delay) */
.sidebar--expanding {
  transition: width 250ms ease 0ms;
}
.sidebar--expanding .nav-label,
.sidebar--expanding .project-label,
.sidebar--expanding .project-name {
  transition: opacity 100ms ease 250ms;
  opacity: 1;
}
/* Text elements default transition */
.nav-label,
.project-label,
.project-name {
  transition: opacity 100ms ease;
}
.sidebar--collapsed {
  width: var(--sidebar-width-collapsed);
}
.sidebar--overlay {
  position: absolute;
  top: 0;
  left: 0;
  bottom: 0;
  width: var(--sidebar-width-expanded);
  z-index: 100;
  box-shadow: var(--shadow-lg);
}
.sidebar-project {
  padding: var(--space-4) var(--space-4) var(--space-2);
  border-bottom: 1px solid var(--color-border-dark);
  min-height: 60px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}
.sidebar--collapsed .sidebar-project {
  align-items: center;
  padding: var(--space-3) 0;
}
.sidebar-favicon {
  width: 24px;
  height: 24px;
  display: block;
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
  position: relative;
}
.badge-dot {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 10px;
  height: 10px;
  padding: 0 2px;
  border-radius: var(--radius-full);
  background: var(--color-error);
  color: #fff;
  font-size: 8px;
  font-weight: 700;
  line-height: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.nav-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}
.sidebar--collapsed .nav-list {
  padding: 0 var(--space-1);
}
.sidebar--collapsed .nav-link {
  justify-content: center;
  padding: var(--space-2);
}
.sidebar--collapsed .nav-label {
  display: none;
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
