<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import QueuePauseBanner from '@/components/queue/QueuePauseBanner.vue'
import QueueRunningPanel from '@/components/queue/QueueRunningPanel.vue'
import QueuePendingTable from '@/components/queue/QueuePendingTable.vue'
import QueueRecentTable from '@/components/queue/QueueRecentTable.vue'
import QueueSidebar from '@/components/queue/QueueSidebar.vue'
import { useQueueStore } from '@/stores/queue'
import { useProjectStore } from '@/stores/project'
import { useRoute, useRouter } from 'vue-router'

const queueStore = useQueueStore()
const projectStore = useProjectStore()
const route = useRoute()
const router = useRouter()

// Active project filter — null means "All Projects"
const activeProject = ref<string | null>(null)

onMounted(() => {
  // Load queue and projects concurrently — neither blocks the other
  void queueStore.fetch()
  void projectStore.fetchProjects().then(() => {
    // After projects load, apply any ?project= query param
    const qp = route.query.project
    const name = typeof qp === 'string' ? qp : null
    if (name && projectStore.projects.some((p) => p.name === name)) {
      activeProject.value = name
    } else if (name) {
      // Unknown project — strip the param so the URL reflects "All Projects".
      const next = { ...route.query }
      delete next.project
      void router.replace({ query: next })
    }
  })
})

function onSidebarSelect(project: string | null) {
  activeProject.value = project
  void router.replace({
    query: project ? { project } : {},
  })
}
</script>

<template>
  <div class="queue-page">
    <AppHeader />
    <div class="queue-body">
      <QueueSidebar v-model="activeProject" @select="onSidebarSelect" />

      <main class="queue-main">
        <div class="queue-content">
          <div class="queue-header">
            <h2 class="queue-title">Work Queue</h2>
            <span v-if="activeProject" class="queue-filter-label">
              Filtered: {{ activeProject }}
            </span>
          </div>

          <div v-if="queueStore.loading" class="state-msg">Loading…</div>

          <template v-else>
            <QueuePauseBanner v-if="queueStore.isPaused" />

            <div class="queue-sections">
              <QueueRunningPanel :project-filter="activeProject" />
              <QueuePendingTable :project-filter="activeProject" />
              <QueueRecentTable :project-filter="activeProject" />
            </div>
          </template>

          <div v-if="queueStore.error" class="state-msg error">{{ queueStore.error }}</div>
        </div>
      </main>
    </div>
  </div>
</template>

<style scoped>
.queue-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}
.queue-body {
  flex: 1;
  display: flex;
  overflow: hidden;
  position: relative;
}
.queue-main {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-8) var(--space-6);
}
.queue-content {
  max-width: 960px;
  margin: 0 auto;
}
.queue-header {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  margin-bottom: var(--space-6);
}
.queue-title {
  font-size: var(--text-2xl);
  font-weight: 700;
  margin: 0;
  color: var(--color-text);
}
.queue-filter-label {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.queue-sections {
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
}
.state-msg {
  padding: var(--space-8) 0;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.state-msg.error { color: #dc2626; }
</style>
