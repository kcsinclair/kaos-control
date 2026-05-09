<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { HelpCircle, AlertTriangle } from 'lucide-vue-next'
import type { ArtifactRow } from '@/types/api'

const props = defineProps<{
  artifact: ArtifactRow
  cardFields: string[]
  age: string
  project: string
  isStale?: boolean
}>()

const router = useRouter()

function open() {
  router.push(`/p/${props.project}/artifacts/${props.artifact.path}`)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') open()
}

const priorityColour: Record<string, string> = {
  critical: '#e53e3e',
  high:     '#dd6b20',
  medium:   '#d69e2e',
  low:      '#38a169',
}

function priorityColor(p: string): string {
  return priorityColour[p.toLowerCase()] ?? '#a0aec0'
}

// When the backend auto-blocks an artifact due to open questions it always
// appends { role: "product-owner", who: "agent" } to assignees.  This is a
// reliable proxy for "blocked due to open questions" without needing the body.
// The kanban board receives live ArtifactRow updates via artifact.indexed
// WebSocket events, so the icon appears/disappears automatically when the
// indexer changes the artifact's status and assignees after an auto-block.
const isBlockedOnQuestions = computed(() =>
  props.artifact.status === 'blocked' &&
  (props.artifact.frontmatter?.assignees ?? []).some(
    (a) => a.role === 'product-owner' && a.who === 'agent',
  ),
)
</script>

<template>
  <div
    class="kanban-card"
    :class="{ 'kanban-card--stale': isStale }"
    tabindex="0"
    role="button"
    :aria-label="`Open artefact: ${artifact.title || artifact.slug}`"
    @click="open"
    @keydown="onKeydown"
  >
    <!-- Title -->
    <div class="card-title">{{ artifact.title || artifact.slug }}</div>

    <!-- Configurable fields -->
    <div class="card-fields">
      <HelpCircle
        v-if="isBlockedOnQuestions"
        class="card-open-questions-icon"
        :size="13"
        title="Blocked pending answers to open questions"
        aria-label="Open questions pending"
      />
      <AlertTriangle
        v-if="isStale"
        class="card-stale-icon"
        :size="13"
        title="QA run may be stuck — in-qa for over 60 minutes"
        aria-label="Stale QA run"
      />
      <span
        v-if="cardFields.includes('type')"
        class="card-badge"
      >{{ artifact.type }}</span>

      <span
        v-if="cardFields.includes('priority') && artifact.frontmatter?.priority"
        class="card-priority"
        :style="{ background: priorityColor(artifact.frontmatter.priority) }"
        :title="artifact.frontmatter.priority"
      >{{ artifact.frontmatter.priority }}</span>

      <template v-if="cardFields.includes('labels') && artifact.frontmatter?.labels?.length">
        <span
          v-for="label in artifact.frontmatter.labels"
          :key="label"
          class="card-label"
        >{{ label }}</span>
      </template>

      <span
        v-if="cardFields.includes('age')"
        class="card-age"
      >{{ age }}</span>
    </div>

    <!-- Lineage slug always shown at bottom -->
    <div class="card-lineage">{{ artifact.lineage }}</div>
  </div>
</template>

<style scoped>
.kanban-card {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  transition: border-color 0.12s, box-shadow 0.12s;
  outline: none;
}
.kanban-card:hover {
  border-color: var(--color-accent);
  box-shadow: 0 1px 4px rgba(0,0,0,0.10);
}
.kanban-card:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}
.card-title {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
  line-height: 1.4;
}
.card-fields {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
  align-items: center;
}
.card-badge {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 500;
  background: var(--color-border);
  color: var(--color-text);
  white-space: nowrap;
}
.card-priority {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  white-space: nowrap;
}
.card-label {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 99px;
  font-size: 10px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
  white-space: nowrap;
}
.card-age {
  font-size: 10px;
  color: var(--color-text-muted);
}
.card-open-questions-icon {
  color: #d97706;
  flex-shrink: 0;
}
.card-stale-icon {
  color: #d97706;
  flex-shrink: 0;
}
.kanban-card--stale {
  border-color: #f59e0b;
}
.kanban-card--stale:hover {
  border-color: #d97706;
}
.card-lineage {
  font-size: 10px;
  color: var(--color-text-muted);
  font-family: monospace;
  margin-top: auto;
}
</style>
