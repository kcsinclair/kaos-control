<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useProjectStore } from '@/stores/project'
import { useQueueStore } from '@/stores/queue'

const props = defineProps<{
  modelValue?: string | null
}>()

const emit = defineEmits<{
  select: [project: string | null]
  'update:modelValue': [project: string | null]
}>()

const projectStore = useProjectStore()
const queueStore = useQueueStore()

const selected = ref<string | null>(props.modelValue ?? null)
const collapsed = ref(false)
const isMobile = ref(false)

// Keep internal selection in sync when parent changes modelValue (e.g. URL param applied after projects load)
watch(() => props.modelValue, (v) => {
  selected.value = v ?? null
})

const mobileQuery = window.matchMedia('(max-width: 767px)')

function applyMedia(matches: boolean) {
  isMobile.value = matches
  if (matches) {
    collapsed.value = true
  }
}

function onMediaChange(e: MediaQueryListEvent) {
  applyMedia(e.matches)
}

onMounted(() => {
  applyMedia(mobileQuery.matches)
  mobileQuery.addEventListener('change', onMediaChange)
  void projectStore.fetchProjects()
})

onUnmounted(() => {
  mobileQuery.removeEventListener('change', onMediaChange)
})

// Count running + pending jobs per project
const jobCounts = computed<Record<string, number>>(() => {
  const counts: Record<string, number> = {}
  const { running, pending } = queueStore.snapshot
  if (running) {
    counts[running.project] = (counts[running.project] ?? 0) + 1
  }
  for (const job of pending) {
    counts[job.project] = (counts[job.project] ?? 0) + 1
  }
  return counts
})

const totalCount = computed(() =>
  Object.values(jobCounts.value).reduce((sum, n) => sum + n, 0),
)

function selectProject(project: string | null) {
  selected.value = project
  emit('select', project)
  emit('update:modelValue', project)
  // Auto-collapse on mobile after selection
  if (isMobile.value) {
    collapsed.value = true
  }
}

function toggleCollapse() {
  collapsed.value = !collapsed.value
}

defineExpose({ selected })
</script>

<template>
  <aside
    class="queue-sidebar"
    :class="{
      'queue-sidebar--collapsed': collapsed,
      'queue-sidebar--mobile': isMobile,
    }"
  >
    <div class="sidebar-header">
      <span v-if="!collapsed" class="sidebar-title">Projects</span>
      <button
        class="collapse-toggle"
        :aria-expanded="String(!collapsed)"
        :aria-label="collapsed ? 'Expand project sidebar' : 'Collapse project sidebar'"
        :aria-controls="'queue-sidebar-nav'"
        @click="toggleCollapse"
      >
        <svg
          v-if="collapsed"
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <!-- PanelLeftOpen icon -->
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
          <line x1="9" y1="3" x2="9" y2="21"/>
          <polyline points="13 8 18 12 13 16"/>
        </svg>
        <svg
          v-else
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <!-- PanelLeftClose icon -->
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
          <line x1="9" y1="3" x2="9" y2="21"/>
          <polyline points="15 8 10 12 15 16"/>
        </svg>
      </button>
    </div>

    <nav
      v-if="!collapsed"
      id="queue-sidebar-nav"
      role="navigation"
      aria-label="Project filter"
      class="sidebar-nav"
    >
      <button
        class="sidebar-item"
        :aria-current="selected === null ? 'page' : undefined"
        :class="{ 'sidebar-item--active': selected === null }"
        @click="selectProject(null)"
      >
        <span class="item-name">All Projects</span>
        <span
          class="item-badge"
          :class="{ 'item-badge--active': totalCount > 0 }"
          aria-label="`${totalCount} active jobs`"
        >{{ totalCount }}</span>
      </button>

      <div v-if="projectStore.loading && !projectStore.projects.length" class="sidebar-loading">
        Loading…
      </div>

      <button
        v-for="project in projectStore.projects"
        :key="project.name"
        class="sidebar-item"
        :aria-current="selected === project.name ? 'page' : undefined"
        :class="{ 'sidebar-item--active': selected === project.name }"
        @click="selectProject(project.name)"
      >
        <span class="item-name">{{ project.name }}</span>
        <span
          class="item-badge"
          :class="{ 'item-badge--active': (jobCounts[project.name] ?? 0) > 0 }"
        >{{ jobCounts[project.name] ?? 0 }}</span>
      </button>
    </nav>
  </aside>
</template>

<style scoped>
.queue-sidebar {
  display: flex;
  flex-direction: column;
  width: 200px;
  min-width: 200px;
  border-right: 1px solid var(--color-border);
  background: var(--color-surface, var(--color-bg));
  flex-shrink: 0;
  transition: width 0.2s ease, min-width 0.2s ease;
  position: relative;
  z-index: 10;
}

.queue-sidebar--collapsed {
  width: 40px;
  min-width: 40px;
}

/* On mobile, overlay instead of occupying layout space */
@media (max-width: 767px) {
  .queue-sidebar--collapsed {
    position: absolute;
    top: 0;
    left: 0;
    bottom: 0;
    width: 40px;
    min-width: 40px;
    z-index: 20;
    border-right: none;
    background: transparent;
  }

  .queue-sidebar--mobile:not(.queue-sidebar--collapsed) {
    position: absolute;
    top: 0;
    left: 0;
    bottom: 0;
    width: 220px;
    min-width: 220px;
    z-index: 20;
    box-shadow: 2px 0 8px rgba(0, 0, 0, 0.15);
  }
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  min-height: 40px;
  gap: var(--space-2);
  background: var(--color-surface, var(--color-bg));
}

.sidebar-title {
  font-size: var(--text-xs, 11px);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
  white-space: nowrap;
  overflow: hidden;
}

.collapse-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text-muted);
  padding: 4px;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
  line-height: 0;
}

.collapse-toggle:hover {
  color: var(--color-text);
  background: var(--color-hover, rgba(0,0,0,0.06));
}

.collapse-toggle:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  padding: var(--space-2) 0;
  overflow-y: auto;
  flex: 1;
  background: var(--color-surface, var(--color-bg));
  border-right: 1px solid var(--color-border);
}

.sidebar-loading {
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}

.sidebar-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  width: 100%;
  padding: var(--space-2) var(--space-3);
  background: none;
  border: none;
  cursor: pointer;
  text-align: left;
  font-size: var(--text-sm);
  color: var(--color-text);
  border-radius: 0;
  transition: background 0.1s;
}

.sidebar-item:hover {
  background: var(--color-hover, rgba(0,0,0,0.06));
}

.sidebar-item:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: -2px;
}

.sidebar-item--active {
  background: var(--color-accent-muted, rgba(99, 102, 241, 0.1));
  color: var(--color-accent);
  font-weight: 500;
}

.sidebar-item--active:hover {
  background: var(--color-accent-muted, rgba(99, 102, 241, 0.15));
}

.item-name {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  border-radius: 9px;
  font-size: 11px;
  font-weight: 600;
  background: var(--color-border);
  color: var(--color-text-muted);
  flex-shrink: 0;
}

.item-badge--active {
  background: var(--color-accent);
  color: #fff;
}
</style>
