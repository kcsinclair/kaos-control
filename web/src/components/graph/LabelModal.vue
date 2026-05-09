<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { GraphNode } from '@/types/api'

const props = defineProps<{
  labelName: string
  project: string
  allNodes: GraphNode[]
}>()

const emit = defineEmits<{ close: [] }>()

const router = useRouter()

const artifacts = computed(() =>
  props.allNodes.filter(
    (n) => n.type !== 'label' && n.labels?.includes(props.labelName)
  )
)

function openArtifact(node: GraphNode) {
  router.push(`/p/${props.project}/artifacts/${node.id}`)
  emit('close')
}

function handleOverlayClick(e: MouseEvent) {
  if ((e.target as HTMLElement).classList.contains('label-modal-overlay')) emit('close')
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div
      class="label-modal-overlay"
      role="dialog"
      aria-modal="true"
      tabindex="-1"
      @click="handleOverlayClick"
      @keydown="handleKeydown"
    >
      <div class="label-modal-panel">
        <button class="label-modal-close" @click="emit('close')" aria-label="Close">✕</button>
        <div class="label-modal-header">
          <span class="label-icon">⬡</span>
          <h2 class="label-modal-title">{{ labelName }}</h2>
          <span class="label-modal-count">{{ artifacts.length }} artifact{{ artifacts.length !== 1 ? 's' : '' }}</span>
        </div>

        <div class="label-modal-body">
          <div v-if="artifacts.length === 0" class="label-empty">No artifacts carry this label.</div>
          <ul v-else class="artifact-list">
            <li
              v-for="node in artifacts"
              :key="node.id"
              class="artifact-item"
              role="button"
              tabindex="0"
              @click="openArtifact(node)"
              @keydown.enter="openArtifact(node)"
            >
              <div class="artifact-item-header">
                <span class="artifact-type">{{ node.type }}</span>
                <span class="artifact-status">{{ node.status }}</span>
              </div>
              <div class="artifact-title">{{ node.title || node.slug }}</div>
              <div class="artifact-path">{{ node.id }}</div>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.label-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  padding: var(--space-6);
}
.label-modal-panel {
  position: relative;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 100%;
  max-width: 480px;
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.label-modal-close {
  position: absolute;
  top: var(--space-4);
  right: var(--space-4);
  background: none;
  border: none;
  font-size: var(--text-lg);
  color: var(--color-text-muted);
  cursor: pointer;
  line-height: 1;
  padding: var(--space-1);
}
.label-modal-close:hover { color: var(--color-text); }
.label-modal-header {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  padding: var(--space-5) var(--space-6) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.label-icon {
  font-size: 14px;
  color: #a855f7;
}
.label-modal-title {
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
  margin: 0;
}
.label-modal-count {
  font-size: 11px;
  color: var(--color-text-muted);
}
.label-modal-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-3) var(--space-4);
}
.label-empty {
  padding: var(--space-4);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  text-align: center;
}
.artifact-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.artifact-item {
  padding: var(--space-3) var(--space-3);
  border-radius: var(--radius-md);
  border: 1px solid transparent;
  cursor: pointer;
  transition: background 0.1s, border-color 0.1s;
}
.artifact-item:hover {
  background: var(--color-surface);
  border-color: var(--color-border);
}
.artifact-item-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: 2px;
}
.artifact-type {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
}
.artifact-status {
  font-size: 10px;
  padding: 0 5px;
  border-radius: 99px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
}
.artifact-title {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
  line-height: 1.3;
}
.artifact-path {
  font-size: 10px;
  font-family: monospace;
  color: var(--color-text-muted);
  margin-top: 1px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
