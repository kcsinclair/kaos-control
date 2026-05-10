<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { useLocksStore } from '@/stores/locks'
import { useAgentsStore } from '@/stores/agents'
import { useSchedulerStore } from '@/stores/scheduler'
import { getProjectWs } from '@/api/ws'
import AppHeader from '@/components/layout/AppHeader.vue'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import InitRequiredBanner from '@/components/project/InitRequiredBanner.vue'

const route = useRoute()
const projectStore = useProjectStore()
const locksStore = useLocksStore()
const agentsStore = useAgentsStore()
const schedulerStore = useSchedulerStore()

const AGENT_EVENTS     = new Set(['agent.started', 'agent.progress', 'agent.finished', 'agent.failed'])
const LOCK_EVENTS      = new Set(['lock.acquired', 'lock.released'])
const SCHEDULER_EVENTS = new Set(['scheduler.job.started', 'scheduler.job.completed'])

let wsUnsub: (() => void) | null = null
let _readyCountDebounce: ReturnType<typeof setTimeout> | null = null

function getProject() { return route.params.project as string }

async function syncProject() {
  const name = getProject()
  if (!projectStore.projects.length) await projectStore.fetchProjects()
  projectStore.setCurrent(name)
  await projectStore.checkInitRequired(name)
}

function scheduleReadyCountRefresh(project: string) {
  if (_readyCountDebounce !== null) clearTimeout(_readyCountDebounce)
  _readyCountDebounce = setTimeout(() => {
    _readyCountDebounce = null
    void agentsStore.fetchReadyCounts(project)
  }, 500)
}

function subscribeWs(project: string) {
  wsUnsub?.()
  const ws = getProjectWs(project)
  wsUnsub = ws.on((e) => {
    if (AGENT_EVENTS.has(e.type)) {
      agentsStore.onWsEvent(e.type, e.payload as Record<string, unknown>)
    } else if (LOCK_EVENTS.has(e.type)) {
      locksStore.applyEvent(e.type, e.payload as Record<string, unknown>)
    } else if (SCHEDULER_EVENTS.has(e.type)) {
      schedulerStore.onWsEvent(e.type, e.payload as Record<string, unknown>)
    } else if (e.type === 'artifact.indexed') {
      scheduleReadyCountRefresh(project)
    }
  })
}

onMounted(() => {
  syncProject()
  subscribeWs(getProject())
})

watch(() => route.params.project, (newProject) => {
  if (newProject) {
    syncProject()
    subscribeWs(newProject as string)
  }
})

onUnmounted(() => {
  wsUnsub?.()
  if (_readyCountDebounce !== null) clearTimeout(_readyCountDebounce)
})
</script>

<template>
  <div class="workspace">
    <AppHeader />
    <div class="workspace-body">
      <AppSidebar />
      <main class="workspace-main">
        <InitRequiredBanner
          v-if="projectStore.initRequired && projectStore.current"
          :path="projectStore.current.path"
        />
        <RouterView v-else />
      </main>
    </div>
  </div>
</template>

<style scoped>
.workspace {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  background: var(--color-bg);
}
.workspace-body {
  flex: 1;
  display: flex;
  overflow: hidden;
  position: relative;
}
.workspace-main {
  flex: 1;
  overflow: auto;
}
</style>
