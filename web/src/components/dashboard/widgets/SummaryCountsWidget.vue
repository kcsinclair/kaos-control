<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api } from '@/api/client'
import { useWebSocket } from '@/composables/useWebSocket'
import { useQueueStore } from '@/stores/queue'
import type { WsEvent } from '@/types/api'
import SummaryCountCard from './SummaryCountCard.vue'
import { Ticket, Play, AlertOctagon, CheckCircle } from 'lucide-vue-next'

const props = defineProps<{ project: string }>()

interface DashboardStats {
  total_tickets: number
  in_progress: number
  blocked: number
  completed_this_week: number
}

const stats = ref<DashboardStats>({ total_tickets: 0, in_progress: 0, blocked: 0, completed_this_week: 0 })
const queueStore = useQueueStore()

async function fetchStats() {
  try {
    const data = await api.get<DashboardStats>(
      `/p/${encodeURIComponent(props.project)}/dashboard/stats`
    )
    stats.value = data
  } catch {
    // keep zeroes on error
  }
}

// Count queue jobs for this project (pending + running) so the "In Progress"
// stat reflects items actively being worked on, not just artifact statuses.
const queueInProgressCount = computed(() => {
  const pending = queueStore.snapshot.pending.filter((j) => j.project === props.project).length
  const running = queueStore.snapshot.running?.project === props.project ? 1 : 0
  return pending + running
})

const inProgressTotal = computed(() => stats.value.in_progress + queueInProgressCount.value)

onMounted(() => {
  void fetchStats()
  void queueStore.fetch()
})

useWebSocket(props.project, 'artifact.indexed', (_e: WsEvent) => {
  void fetchStats()
})
</script>

<template>
  <SummaryCountCard
    label="Lifecycle Total"
    :value="stats.total_tickets"
    :icon="Ticket"
    :to="{ name: 'artifacts', params: { project: props.project }, query: {} }"
  />
  <SummaryCountCard
    label="In Progress"
    :value="inProgressTotal"
    :icon="Play"
    :to="null"
  />
  <SummaryCountCard
    label="Blocked"
    :value="stats.blocked"
    :icon="AlertOctagon"
    :to="{ name: 'artifacts', params: { project: props.project }, query: { status: 'blocked' } }"
  />
  <SummaryCountCard
    label="Completed This Week"
    :value="stats.completed_this_week"
    :icon="CheckCircle"
    :to="null"
  />
</template>
