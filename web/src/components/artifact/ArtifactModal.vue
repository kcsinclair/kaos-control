<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useArtifactsStore } from '@/stores/artifacts'
import MarkdownPreview from './MarkdownPreview.vue'
import TransitionDialog from './TransitionDialog.vue'
import RunAgentDialog from '@/components/agent/RunAgentDialog.vue'
import type { GraphNode, ArtifactDetail, GraphEdge } from '@/types/api'

const props = defineProps<{
  node: GraphNode | null
  project: string
  edges: GraphEdge[]
}>()

const emit = defineEmits<{ close: [] }>()

const router = useRouter()
const store = useArtifactsStore()

const detail = ref<ArtifactDetail | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const showTransition = ref(false)
const showRunAgent = ref(false)

function onTransitioned(newStatus: string) {
  showTransition.value = false
  // Update the local status display optimistically
  if (detail.value) detail.value = { ...detail.value, status: newStatus }
}

function onRunStarted() {
  showRunAgent.value = false
  router.push(`/p/${props.project}/agents`)
  emit('close')
}

// Edges for the current node
const inbound = computed(() =>
  props.edges.filter((e) => e.target === props.node?.id)
)
const outbound = computed(() =>
  props.edges.filter((e) => e.source === props.node?.id)
)

watch(
  () => props.node,
  async (node) => {
    if (!node) { detail.value = null; return }
    loading.value = true
    error.value = null
    detail.value = null
    try {
      detail.value = await store.fetchOne(props.project, node.id)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to load'
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

function openEditor() {
  if (!props.node) return
  router.push(`/p/${props.project}/artifacts/${props.node.id}`)
  emit('close')
}

function handleOverlayClick(e: MouseEvent) {
  if ((e.target as HTMLElement).classList.contains('modal-overlay')) emit('close')
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

const STATUS_COLORS: Record<string, string> = {
  done: '#d1fae5',
  approved: '#dbeafe',
  'in-progress': '#fef3c7',
  blocked: '#fee2e2',
}
const STATUS_TEXT: Record<string, string> = {
  done: '#065f46',
  approved: '#1e40af',
  'in-progress': '#92400e',
  blocked: '#991b1b',
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="node"
      class="modal-overlay"
      role="dialog"
      aria-modal="true"
      tabindex="-1"
      @click="handleOverlayClick"
      @keydown="handleKeydown"
    >
      <div class="modal-panel">
        <div class="modal-header">
          <div class="modal-meta">
            <span class="modal-type">{{ node.type }}</span>
            <span
              class="modal-status"
              :style="{ background: STATUS_COLORS[node.status] ?? '#e5e7eb', color: STATUS_TEXT[node.status] ?? '#374151' }"
            >{{ node.status }}</span>
          </div>
          <h2 class="modal-title">{{ node.title || node.slug }}</h2>
          <div class="modal-path">{{ node.id }}</div>
        </div>

        <div class="modal-actions">
          <button class="action-btn action-btn--primary" @click="openEditor">Edit</button>
          <button class="action-btn" @click="showTransition = true">Change Status</button>
          <button class="action-btn" @click="showRunAgent = true">Run Agent</button>
        </div>

        <div class="modal-body">
          <div v-if="loading" class="state-msg">Loading…</div>
          <div v-else-if="error" class="state-msg error">{{ error }}</div>
          <MarkdownPreview
            v-else-if="detail"
            :html="detail.body_html"
            :source="detail.body"
            :project="project"
          />
          <div v-else class="state-msg">No preview available.</div>
        </div>

        <div class="modal-footer" v-if="inbound.length || outbound.length">
          <div v-if="outbound.length" class="edge-group">
            <div class="edge-group-label">Outbound</div>
            <div v-for="e in outbound" :key="e.target + e.kind" class="edge-item">
              <span class="edge-kind">{{ e.kind }}</span>
              <span class="edge-path">{{ e.target }}</span>
            </div>
          </div>
          <div v-if="inbound.length" class="edge-group">
            <div class="edge-group-label">Inbound</div>
            <div v-for="e in inbound" :key="e.source + e.kind" class="edge-item">
              <span class="edge-kind">{{ e.kind }}</span>
              <span class="edge-path">{{ e.source }}</span>
            </div>
          </div>
        </div>

        <button class="modal-close" @click="emit('close')" aria-label="Close">✕</button>
      </div>
    </div>

    <TransitionDialog
      v-if="showTransition && node"
      :project="project"
      :path="node.id"
      :current-status="detail?.status ?? node.status"
      @transitioned="onTransitioned"
      @cancel="showTransition = false"
    />

    <RunAgentDialog
      v-if="showRunAgent && node"
      :project="project"
      :target-path="node.id"
      @started="onRunStarted"
      @cancel="showRunAgent = false"
    />
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  padding: var(--space-6);
}
.modal-panel {
  position: relative;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 100%;
  max-width: 680px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-header {
  padding: var(--space-5) var(--space-6) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.modal-meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-2);
}
.modal-type {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
}
.modal-status {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
}
.modal-title {
  font-size: var(--text-xl);
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 var(--space-1);
  line-height: 1.3;
}
.modal-path {
  font-size: 11px;
  font-family: monospace;
  color: var(--color-text-muted);
}
.modal-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.action-btn {
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-sm);
  border: none;
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.action-btn--primary {
  background: var(--color-accent);
  color: #fff;
}
.action-btn--primary:hover { opacity: 0.88; }
.action-btn:not(.action-btn--primary) {
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
}
.action-btn:not(.action-btn--primary):hover { background: var(--color-border); }
.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-6);
}
.state-msg {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.state-msg.error { color: #dc2626; }
.modal-footer {
  border-top: 1px solid var(--color-border);
  padding: var(--space-4) var(--space-6);
  display: flex;
  gap: var(--space-6);
  flex-shrink: 0;
  overflow-x: auto;
}
.edge-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
}
.edge-group-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-muted);
  margin-bottom: 2px;
}
.edge-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: 11px;
}
.edge-kind {
  color: var(--color-text-muted);
  flex-shrink: 0;
}
.edge-path {
  font-family: monospace;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.modal-close {
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
.modal-close:hover { color: var(--color-text); }
</style>
