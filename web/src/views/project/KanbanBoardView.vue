<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useKanbanBoard } from '@/composables/useKanbanBoard'
import KanbanCard from '@/components/artifact/KanbanCard.vue'

const route = useRoute()
const project = route.params.project as string

const {
  loading,
  hasConfig,
  columns,
  cardFields,
  refresh,
  ageOf,
} = useKanbanBoard(project)

onMounted(refresh)
</script>

<template>
  <div class="board-view">
    <div class="board-header">
      <h2 class="board-title">Board</h2>
    </div>

    <div v-if="loading" class="state-msg">Loading…</div>
    <div v-else-if="!hasConfig" class="state-msg">
      No Kanban configuration found. Add a <code>kanban</code> section to your project's config.yaml.
    </div>

    <!-- Board -->
    <div v-else class="board-columns">
      <div
        v-for="col in columns"
        :key="col.name"
        class="board-column"
        role="region"
        :aria-label="col.name"
      >
        <div class="column-header">
          <span class="column-name">{{ col.name }}</span>
          <span class="column-count">{{ col.cards.length }}</span>
        </div>
        <div class="column-cards">
          <div v-if="col.cards.length === 0" class="column-empty">No artefacts</div>
          <KanbanCard
            v-for="card in col.cards"
            :key="card.path"
            :artifact="card"
            :card-fields="cardFields"
            :age="ageOf(card)"
            :project="project"
          />
        </div>
      </div>
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
.board-columns {
  display: flex;
  flex: 1;
  overflow-x: auto;
  overflow-y: hidden;
  gap: var(--space-4);
  padding: var(--space-4) var(--space-6);
  align-items: flex-start;
}
.board-column {
  display: flex;
  flex-direction: column;
  min-width: 280px;
  max-width: 280px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  flex-shrink: 0;
  height: 100%;
  overflow: hidden;
}
.column-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.column-name {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}
.column-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--radius-full);
  background: var(--color-border);
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-muted);
}
.column-cards {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
}
.column-empty {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  text-align: center;
  padding: var(--space-4) 0;
}
</style>
