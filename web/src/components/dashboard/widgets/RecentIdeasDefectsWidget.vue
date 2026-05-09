<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listArtifacts } from '@/api/artifacts'
import { useWebSocket } from '@/composables/useWebSocket'
import type { ArtifactRow, WsEvent } from '@/types/api'

const props = defineProps<{ project: string }>()

const items = ref<ArtifactRow[]>([])
const loading = ref(false)

async function fetchItems() {
  loading.value = true
  try {
    const data = await listArtifacts(props.project, {
      type: 'idea,defect',
      sort: 'created:desc',
      limit: 6,
    })
    items.value = data.items ?? []
  } catch {
    items.value = []
  } finally {
    loading.value = false
  }
}

onMounted(fetchItems)

useWebSocket(props.project, 'artifact.indexed', (_e: WsEvent) => {
  void fetchItems()
})

function relativeTime(iso: string): string {
  const diff = (Date.now() - new Date(iso).getTime()) / 1000
  const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' })
  if (diff < 60) return rtf.format(-Math.round(diff), 'second')
  if (diff < 3600) return rtf.format(-Math.round(diff / 60), 'minute')
  if (diff < 86400) return rtf.format(-Math.round(diff / 3600), 'hour')
  if (diff < 2592000) return rtf.format(-Math.round(diff / 86400), 'day')
  return rtf.format(-Math.round(diff / 2592000), 'month')
}
</script>

<template>
  <div class="recent-ideas-defects-widget">
    <div class="widget-header">
      <h3 class="widget-title">Recent Ideas &amp; Defects</h3>
    </div>

    <ul v-if="items.length" class="item-list" role="list">
      <li v-for="item in items" :key="item.path" class="item" role="listitem">
        <router-link
          :to="'/p/' + project + '/artifacts/' + item.path"
          class="item-link"
        >
          <span
            :class="['type-badge', 'type-badge--' + item.type]"
            :aria-label="'Type: ' + item.type"
          >{{ item.type }}</span>
          <span class="item-title">{{ item.title }}</span>
          <span class="item-time">{{ relativeTime(item.created) }}</span>
        </router-link>
      </li>
    </ul>
    <div v-else-if="loading" class="empty-state">Loading…</div>
    <div v-else class="empty-state">No recent ideas or defects</div>
  </div>
</template>

<style scoped>
.recent-ideas-defects-widget {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-width: 0;
  /* match the total visual height of the adjacent pie chart widgets */
  min-height: 320px;
}

.widget-header {
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

.item-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.item {
  border-radius: var(--radius-md);
}

.item-link {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-2);
  border-radius: var(--radius-md);
  text-decoration: none;
  color: var(--color-text);
  min-width: 0;
  transition: background 0.1s;
}

.item-link:hover {
  background: var(--color-surface-raised, rgba(0 0 0 / 0.04));
}

.item-link:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

/* Type badge — idea: blue-tinted, defect: amber-tinted */
.type-badge {
  flex-shrink: 0;
  display: inline-block;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  padding: 2px 6px;
  border-radius: 4px;
  line-height: 1.4;
}

/* idea: dark text on light-blue bg — contrast ≥ 4.5:1 */
.type-badge--idea {
  background: #dbeafe;
  color: #1e40af;
}

/* defect: dark text on light-amber bg — contrast ≥ 4.5:1 */
.type-badge--defect {
  background: #fef3c7;
  color: #92400e;
}

.item-title {
  flex: 1;
  font-size: var(--text-sm);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.item-time {
  flex-shrink: 0;
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  white-space: nowrap;
}

.empty-state {
  text-align: center;
  padding: var(--space-6) 0;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
</style>
