<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { useTestingStore } from '@/stores/testing'
import { useAppStore } from '@/stores/app'
import { api } from '@/api/client'
import { useWebSocket } from '@/composables/useWebSocket'
import type { WsEvent } from '@/types/api'
import {
  ChevronLeft,
  ChevronRight,
  LayoutDashboard,
  List,
  Columns3,
  Network,
  Bot,
  Activity,
  AlertTriangle,
  Settings,
  Layers,
  CalendarClock,
  CalendarRange,
  Server,
  FlaskConical,
  ListChecks,
  BarChart3,
} from 'lucide-vue-next'
import type { Component } from 'vue'
import SidebarTooltip from '@/components/ui/SidebarTooltip.vue'
import GitStatusBar from '@/components/layout/GitStatusBar.vue'

const route = useRoute()
const projectStore = useProjectStore()
const uiStore = useUiStore()
const authStore = useAuthStore()
const testingStore = useTestingStore()
const appStore = useAppStore()

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

onMounted(() => {
  const p = projectName()
  fetchParseErrors(p)
  testingStore.fetchApprovedCount(p)
})

watch(() => route.params.project, (p) => {
  if (p) {
    fetchParseErrors(p as string)
    testingStore.fetchApprovedCount(p as string)
  }
})

// Re-check when artifacts are re-indexed (a re-index may fix or introduce errors)
useWebSocket(projectName(), 'artifact.indexed', (_e: WsEvent) => {
  fetchParseErrors(projectName())
  testingStore.fetchApprovedCount(projectName())
})

interface NavItem {
  label: string
  to: string
  icon: Component
  roles?: string[]
  badgeCount?: () => number
}

const navItems = computed((): NavItem[] => {
  const p = projectName()
  const roles = authStore.rolesForProject(p)
  const hasDevOpsAccess = roles.includes('product-owner') || roles.includes('devops')
  const items: NavItem[] = [
    { label: 'Dashboard',    to: `/p/${p}/dashboard`,       icon: LayoutDashboard },
    { label: 'List',         to: `/p/${p}/artifacts`,       icon: List },
    { label: 'Board',        to: `/p/${p}/artifacts/board`, icon: Columns3 },
    { label: 'Testing',      to: `/p/${p}/testing`,         icon: FlaskConical, badgeCount: () => testingStore.approvedCount },
    { label: 'Map',          to: `/p/${p}/map`,             icon: Network },
    { label: 'Roadmap',      to: `/p/${p}/roadmap`,         icon: CalendarRange },
    { label: 'Agents',       to: `/p/${p}/agents`,          icon: Bot },
    { label: 'Reports',      to: `/p/${p}/reports`,         icon: BarChart3 },
    // Queue is a global route (not project-scoped) but lives in the project
    // sidebar for discoverability — it's the natural place users look when
    // they've queued work from an artefact view.
    { label: 'Queue',        to: `/queue`,                  icon: ListChecks },
    { label: 'Scheduler',    to: `/p/${p}/scheduler`,       icon: CalendarClock },
    { label: 'Feed',         to: `/p/${p}/feed`,            icon: Activity },
    { label: 'Parse Errors', to: `/p/${p}/parse-errors`,    icon: AlertTriangle },
    { label: 'Config',       to: `/p/${p}/config`,          icon: Settings },
    { label: 'Ollama',       to: `/p/${p}/settings/ollama`, icon: Server },
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

// Mobile drawer: close on route change so tapping a nav link dismisses the
// drawer instead of leaving it sitting over the new view. Watching params
// alone (e.g. project) misses sibling-route navigation; watch the full path.
watch(
  () => route.fullPath,
  () => {
    if (uiStore.mobileSidebarOpen) uiStore.closeMobileSidebar()
  }
)

// ESC key closes the drawer.
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && uiStore.mobileSidebarOpen) uiStore.closeMobileSidebar()
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <!-- Mobile backdrop: covers the main content area while the drawer is open
       so taps outside the drawer dismiss it. Only present at ≤640 px via CSS. -->
  <div
    v-if="uiStore.mobileSidebarOpen"
    class="sidebar-backdrop"
    role="presentation"
    @click="uiStore.closeMobileSidebar()"
  />
  <nav
    class="app-sidebar"
    :class="{
      'sidebar--collapsed': uiStore.sidebarCollapsed && !hoverExpanded,
      'sidebar--overlay': uiStore.sidebarCollapsed && hoverExpanded,
      'sidebar--expanding': sidebarTransitionDir === 'expanding',
      'sidebar--collapsing': sidebarTransitionDir === 'collapsing',
      'sidebar--mobile-open': uiStore.mobileSidebarOpen,
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
              <!-- Collapsed dot badge: parse errors -->
              <span
                v-if="item.label === 'Parse Errors' && parseErrorCount > 0 && !isVisuallyExpanded()"
                class="badge-dot"
                :aria-label="`${parseErrorCount} parse errors`"
              >{{ parseErrorCount > 9 ? parseErrorCount : '' }}</span>
              <!-- Collapsed dot badge: generic badgeCount -->
              <span
                v-else-if="item.badgeCount && item.badgeCount() > 0 && !isVisuallyExpanded()"
                class="badge-dot"
                :aria-label="`${item.badgeCount()} ${item.label.toLowerCase()}`"
              ></span>
            </span>
            <span class="nav-label">{{ item.label }}</span>
            <!-- Expanded badge: parse errors -->
            <span
              v-if="item.label === 'Parse Errors' && parseErrorCount > 0 && isVisuallyExpanded()"
              class="badge"
              :aria-label="`${parseErrorCount} parse errors`"
            >{{ parseErrorCount }}</span>
            <!-- Expanded badge: generic badgeCount -->
            <span
              v-else-if="item.badgeCount && item.badgeCount() > 0 && isVisuallyExpanded()"
              class="badge"
              :aria-label="`${item.badgeCount()} ${item.label.toLowerCase()}`"
            >{{ item.badgeCount() }}</span>
          </RouterLink>
        </SidebarTooltip>
      </li>
    </ul>
    <!-- Git status panel -->
    <GitStatusBar :project="projectName()" :collapsed="uiStore.sidebarCollapsed && !hoverExpanded" />
    <div class="sidebar-version" aria-label="Application version">
      <span v-if="isVisuallyExpanded()" class="version-label">kaos-control {{ appStore.version }}</span>
    </div>
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
.sidebar-version {
  padding: var(--space-2) var(--space-4);
  min-height: 24px;
  display: flex;
  align-items: center;
}
.version-label {
  font-size: 0.75rem;
  color: var(--color-sidebar-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  user-select: none;
}

/* ─── Mobile drawer ───────────────────────────────────────────────────────
   On ≤640px the sidebar is hidden by default and slides in from the left
   as an overlay when uiStore.mobileSidebarOpen is true. The persistent
   sidebar pattern is bad on phones: a 220px sidebar steals 58% of a 375px
   viewport even when "collapsed".
   ─────────────────────────────────────────────────────────────────────── */
.sidebar-backdrop {
  display: none;
}
@media (max-width: 640px) {
  .app-sidebar {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: min(280px, 85vw);
    /* Always render at expanded width in drawer mode; the desktop --collapsed
       class is irrelevant on mobile because the drawer is hidden by default. */
    transform: translateX(-100%);
    transition: transform 220ms ease;
    z-index: var(--z-sidebar);
    box-shadow: var(--shadow-lg);
  }
  .app-sidebar.sidebar--collapsed {
    /* Override desktop collapse width — we want full text labels in the drawer. */
    width: min(280px, 85vw);
  }
  .app-sidebar.sidebar--mobile-open {
    transform: translateX(0);
  }
  /* Force the labels visible in the drawer regardless of the desktop
     collapsed-fade animations. */
  .app-sidebar.sidebar--mobile-open .nav-label,
  .app-sidebar.sidebar--mobile-open .project-label,
  .app-sidebar.sidebar--mobile-open .project-name {
    opacity: 1;
  }
  /* Hide the desktop collapse toggle on mobile — it's meaningless when the
     drawer's open/closed state is what matters. */
  .app-sidebar .sidebar-toggle {
    display: none;
  }
  .sidebar-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    z-index: calc(var(--z-sidebar) - 1);
    animation: sidebar-backdrop-in 180ms ease;
  }
  @keyframes sidebar-backdrop-in {
    from { opacity: 0; }
    to   { opacity: 1; }
  }
}
@media (prefers-reduced-motion: reduce) {
  .app-sidebar { transition: none; }
  .sidebar-backdrop { animation: none; }
}
</style>
