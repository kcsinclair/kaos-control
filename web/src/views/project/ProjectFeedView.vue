<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useFeedStore } from '@/stores/feed'
import { useWebSocket } from '@/composables/useWebSocket'
import type { WsEvent } from '@/types/api'
import FeedEntry from '@/components/feed/FeedEntry.vue'
import FeedFilterBar from '@/components/feed/FeedFilterBar.vue'

const route = useRoute()
const router = useRouter()
const feedStore = useFeedStore()

const project = computed(() => route.params.project as string)

// Track IDs of events prepended via WebSocket so FeedEntry can animate them
const newEventIds = ref(new Set<number>())

// Infinite scroll sentinel
const sentinelRef = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null

// Keyboard navigation
const feedListRef = ref<HTMLOListElement | null>(null)
const focusedIndex = ref(-1)

function handleKeydown(e: KeyboardEvent) {
  const items = feedListRef.value?.querySelectorAll<HTMLElement>('.feed-entry-item')
  if (!items || items.length === 0) return

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    focusedIndex.value = Math.min(focusedIndex.value + 1, items.length - 1)
    items[focusedIndex.value]?.focus()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    focusedIndex.value = Math.max(focusedIndex.value - 1, 0)
    items[focusedIndex.value]?.focus()
  } else if (e.key === 'Enter') {
    const link = items[focusedIndex.value]?.querySelector<HTMLAnchorElement>('a')
    if (link) {
      const href = link.getAttribute('href')
      if (href) router.push(href)
    }
  }
}

onMounted(async () => {
  await feedStore.refresh(project.value)

  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0]?.isIntersecting && feedStore.nextCursor !== null && !feedStore.loading) {
        void feedStore.loadPage(project.value)
      }
    },
    { threshold: 0.1 },
  )

  if (sentinelRef.value) {
    observer.observe(sentinelRef.value)
  }
})

onUnmounted(() => {
  observer?.disconnect()
})

// Real-time feed events — prepend and briefly mark as new for animation
useWebSocket(project.value, 'feed.new', (e: WsEvent) => {
  const event = e.payload as import('@/types/api').FeedEvent
  feedStore.prepend(event)
  newEventIds.value.add(event.id)
  setTimeout(() => {
    newEventIds.value.delete(event.id)
  }, 1200)
})
</script>

<template>
  <div class="feed-view">
    <header class="feed-header">
      <h2 class="feed-title">Activity Feed</h2>
      <FeedFilterBar
        :active-types="feedStore.activeTypes"
        @toggle="feedStore.setFilter"
      />
    </header>

    <ol
      v-if="feedStore.events.length > 0"
      ref="feedListRef"
      class="feed-list"
      role="list"
      @keydown="handleKeydown"
    >
      <li
        v-for="event in feedStore.events"
        :key="event.id"
        class="feed-entry-item"
        tabindex="0"
        role="listitem"
      >
        <FeedEntry
          :event="event"
          :project="project"
          :is-new="newEventIds.has(event.id)"
        />
      </li>
    </ol>

    <div
      v-if="feedStore.events.length === 0 && !feedStore.loading"
      class="feed-empty"
    >
      No activity yet
    </div>

    <div v-if="feedStore.loading" class="feed-loading">Loading…</div>

    <!-- Infinite scroll sentinel -->
    <div ref="sentinelRef" class="feed-sentinel" aria-hidden="true" />
  </div>
</template>

<style scoped>
.feed-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: 900px;
  margin: 0 auto;
  padding: var(--space-6) var(--space-4);
  box-sizing: border-box;
}

@media (max-width: 1023px) {
  .feed-view {
    max-width: 100%;
    padding: var(--space-4) var(--space-3);
  }
}

.feed-header {
  margin-bottom: var(--space-4);
}

.feed-title {
  font-size: var(--text-xl);
  font-weight: 600;
  margin: 0 0 var(--space-3) 0;
  color: var(--color-text);
}

.feed-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
  flex: 1;
}

.feed-entry-item {
  border-radius: var(--radius-md);
  outline: none;
}

.feed-entry-item:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.feed-empty,
.feed-loading {
  text-align: center;
  padding: var(--space-8) 0;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}

.feed-sentinel {
  height: 1px;
  margin-top: var(--space-4);
}
</style>
