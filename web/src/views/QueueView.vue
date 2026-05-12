<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { onMounted } from 'vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import QueuePauseBanner from '@/components/queue/QueuePauseBanner.vue'
import QueueRunningPanel from '@/components/queue/QueueRunningPanel.vue'
import QueuePendingTable from '@/components/queue/QueuePendingTable.vue'
import QueueRecentTable from '@/components/queue/QueueRecentTable.vue'
import { useQueueStore } from '@/stores/queue'

const queueStore = useQueueStore()

onMounted(() => {
  void queueStore.fetch()
})
</script>

<template>
  <div class="queue-page">
    <AppHeader />
    <main class="queue-main">
      <div class="queue-content">
        <div class="queue-header">
          <h2 class="queue-title">Work Queue</h2>
        </div>

        <div v-if="queueStore.loading" class="state-msg">Loading…</div>

        <template v-else>
          <QueuePauseBanner v-if="queueStore.isPaused" />

          <div class="queue-sections">
            <QueueRunningPanel />
            <QueuePendingTable />
            <QueueRecentTable />
          </div>
        </template>

        <div v-if="queueStore.error" class="state-msg error">{{ queueStore.error }}</div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.queue-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
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
  margin-bottom: var(--space-6);
}
.queue-title {
  font-size: var(--text-2xl);
  font-weight: 700;
  margin: 0;
  color: var(--color-text);
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
