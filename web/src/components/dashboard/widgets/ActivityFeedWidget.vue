<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { fetchFeed } from '@/api/feed'
import { useWebSocket } from '@/composables/useWebSocket'
import type { FeedEvent, WsEvent } from '@/types/api'
import FeedEntry from '@/components/feed/FeedEntry.vue'

const props = defineProps<{ project: string }>()

const router = useRouter()
const events = ref<FeedEvent[]>([])
const newEventIds = ref(new Set<number>())
const loading = ref(false)

async function fetchEvents() {
  loading.value = true
  try {
    const data = await fetchFeed(props.project, { limit: 7 })
    events.value = data.events ?? []
  } catch {
    events.value = []
  } finally {
    loading.value = false
  }
}

onMounted(fetchEvents)

useWebSocket(props.project, 'feed.new', (e: WsEvent) => {
  const event = e.payload as FeedEvent
  events.value.unshift(event)
  if (events.value.length > 7) events.value.splice(7)
  newEventIds.value.add(event.id)
  setTimeout(() => {
    newEventIds.value.delete(event.id)
  }, 1200)
})

function viewAll() {
  void router.push(`/p/${props.project}/feed`)
}
</script>

<template>
  <div class="activity-feed-widget">
    <div class="activity-feed-header">
      <h3 class="widget-title">Recent Activity</h3>
      <button class="view-all-btn" @click="viewAll">View all</button>
    </div>

    <div class="activity-feed-body">
      <ol v-if="events.length" class="feed-list" role="list">
        <li
          v-for="event in events"
          :key="event.id"
          class="feed-list-item"
          role="listitem"
        >
          <FeedEntry
            :event="event"
            :project="project"
            :is-new="newEventIds.has(event.id)"
          />
        </li>
      </ol>
      <div v-else-if="loading" class="feed-empty">Loading…</div>
      <div v-else class="feed-empty">No activity yet</div>
    </div>
  </div>
</template>

<style scoped>
.activity-feed-widget {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-width: 0;
}

.activity-feed-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.widget-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.view-all-btn {
  font-size: var(--text-xs);
  color: var(--color-primary);
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.view-all-btn:hover,
.view-all-btn:focus-visible {
  opacity: 0.8;
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.activity-feed-body {
  overflow-y: auto;
  max-height: 560px;
}

.feed-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.feed-list-item {
  border-radius: var(--radius-md);
}

.feed-empty {
  text-align: center;
  padding: var(--space-6) 0;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
</style>
