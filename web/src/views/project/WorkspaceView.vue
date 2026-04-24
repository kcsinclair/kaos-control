<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { useLocksStore } from '@/stores/locks'
import { useAgentsStore } from '@/stores/agents'
import { getProjectWs } from '@/api/ws'
import AppHeader from '@/components/layout/AppHeader.vue'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import RunStatusChip from '@/components/agent/RunStatusChip.vue'

const route = useRoute()
const projectStore = useProjectStore()
const locksStore = useLocksStore()
const agentsStore = useAgentsStore()

const AGENT_EVENTS = new Set(['agent.started', 'agent.progress', 'agent.finished', 'agent.failed'])
const LOCK_EVENTS  = new Set(['lock.acquired', 'lock.released'])

let wsUnsub: (() => void) | null = null

function getProject() { return route.params.project as string }

async function syncProject() {
  const name = getProject()
  if (!projectStore.projects.length) await projectStore.fetchProjects()
  projectStore.setCurrent(name)
}

function subscribeWs(project: string) {
  wsUnsub?.()
  const ws = getProjectWs(project)
  wsUnsub = ws.on((e) => {
    if (AGENT_EVENTS.has(e.type)) {
      agentsStore.onWsEvent(e.type, e.payload as Record<string, unknown>)
    } else if (LOCK_EVENTS.has(e.type)) {
      locksStore.applyEvent(e.type, e.payload as Record<string, unknown>)
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

onUnmounted(() => { wsUnsub?.() })
</script>

<template>
  <div class="workspace">
    <AppHeader />
    <div class="workspace-body">
      <AppSidebar />
      <main class="workspace-main">
        <RouterView />
      </main>
    </div>
    <RunStatusChip :project="getProject()" />
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
}
.workspace-main {
  flex: 1;
  overflow: auto;
}
</style>
