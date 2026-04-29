<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { FeedEvent } from '@/types/api'
import {
  ArrowRightLeft,
  FilePlus,
  Play,
  CheckCircle,
  XCircle,
  Bug,
  GitCommit,
  Activity,
} from 'lucide-vue-next'
import type { Component } from 'vue'

const props = defineProps<{
  event: FeedEvent
  project: string
}>()

const router = useRouter()

function iconForType(type: string): Component {
  const map: Record<string, Component> = {
    status_transition: ArrowRightLeft,
    artifact_created: FilePlus,
    agent_started: Play,
    agent_finished: CheckCircle,
    agent_failed: XCircle,
    defect_raised: Bug,
    git_committed: GitCommit,
  }
  return map[type] ?? Activity
}

const isoTimestamp = computed(() =>
  new Date(props.event.timestamp * 1000).toISOString(),
)

const relativeTime = computed(() => {
  const now = Date.now()
  const diff = Math.floor((now - props.event.timestamp * 1000) / 1000)
  if (diff < 60) return `${diff}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)} min ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)} hours ago`
  return `${Math.floor(diff / 86400)} days ago`
})

const navigationTarget = computed(() => {
  if (props.event.artifact_path) {
    return `/p/${props.project}/artifacts/${props.event.artifact_path}`
  }
  if (props.event.run_id) {
    return `/p/${props.project}/agents`
  }
  return router.currentRoute.value.fullPath
})
</script>

<template>
  <router-link :to="navigationTarget" class="feed-entry-link">
    <span class="feed-icon" :aria-label="event.event_type">
      <component :is="iconForType(event.event_type)" :size="16" />
    </span>
    <time class="feed-time" :datetime="isoTimestamp">{{ relativeTime }}</time>
    <span class="feed-summary">{{ event.summary }}</span>
    <span class="feed-actor">{{ event.actor }}</span>
  </router-link>
</template>

<style scoped>
.feed-entry-link {
  display: grid;
  grid-template-columns: 24px 80px 1fr auto;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  text-decoration: none;
  color: var(--color-text);
  background: var(--color-surface);
  transition: background 0.12s;
  min-height: 36px;
}

.feed-entry-link:hover {
  background: var(--color-surface-hover, var(--color-sidebar-hover));
}

.feed-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  flex-shrink: 0;
}

.feed-time {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  white-space: nowrap;
  flex-shrink: 0;
}

.feed-summary {
  font-size: var(--text-sm);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.feed-actor {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  white-space: nowrap;
  flex-shrink: 0;
}

@keyframes feed-entry-highlight {
  from { background: color-mix(in srgb, var(--color-primary) 15%, transparent); }
  to   { background: var(--color-surface); }
}

.feed-entry-link.feed-entry--new {
  animation: feed-entry-highlight 1s ease-out forwards;
}
</style>
