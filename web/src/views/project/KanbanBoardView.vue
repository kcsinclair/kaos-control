<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'

interface KanbanColumn {
  name: string
  statuses: string[]
}

interface KanbanConfig {
  columns: KanbanColumn[]
  uncategorised?: boolean
  card_fields?: string[]
}

const route = useRoute()
const project = route.params.project as string

const loading = ref(true)
const kanban = ref<KanbanConfig | null>(null)

async function fetchKanbanConfig() {
  loading.value = true
  try {
    const res = await api.get<{ kanban: KanbanConfig | null }>(
      `/p/${encodeURIComponent(project)}/config/kanban`
    )
    kanban.value = res.kanban
  } catch {
    kanban.value = null
  } finally {
    loading.value = false
  }
}

onMounted(fetchKanbanConfig)
</script>

<template>
  <div class="board-view">
    <div class="board-header">
      <h2 class="board-title">Board</h2>
    </div>

    <div v-if="loading" class="state-msg">Loading…</div>
    <div v-else-if="!kanban" class="state-msg">
      No Kanban configuration found. Add a <code>kanban</code> section to your project's config.yaml.
    </div>
    <div v-else class="board-placeholder">
      <!-- columns rendered in Milestone 5 -->
    </div>
  </div>
</template>

<style scoped>
.board-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.board-header {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.board-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.state-msg {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.board-placeholder {
  flex: 1;
}
</style>
