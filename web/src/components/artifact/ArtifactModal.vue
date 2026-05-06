<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useArtifactsStore } from '@/stores/artifacts'
import { useGraphStore } from '@/stores/graph'
import { patchPriority } from '@/api/artifacts'
import MarkdownPreview from './MarkdownPreview.vue'
import TransitionDialog from './TransitionDialog.vue'
import RunAgentDialog from '@/components/agent/RunAgentDialog.vue'
import ArtifactRunHistory from './ArtifactRunHistory.vue'
import RunDetailModal from '@/components/agent/RunDetailModal.vue'
import { PRIORITY_COLORS } from '@/components/graph/graphConstants'
import StatusCheckPanel from './StatusCheckPanel.vue'
import type { GraphNode, ArtifactDetail, GraphEdge } from '@/types/api'

const props = defineProps<{
  node: GraphNode | null
  project: string
  edges: GraphEdge[]
}>()

const emit = defineEmits<{ close: [] }>()

const router = useRouter()
const store = useArtifactsStore()
const graphStore = useGraphStore()

const detail = ref<ArtifactDetail | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const showTransition = ref(false)
const showRunAgent = ref(false)
const priorityEditing = ref(false)
const priorityError = ref<string | null>(null)
const selectedRunId = ref<string | null>(null)
const showStatusPanel = ref(false)
const statusCheckLoading = ref(false)

const PRIORITY_OPTIONS = ['high', 'medium', 'normal', 'low']

async function onPriorityChange(e: Event) {
  if (!props.node) return
  const value = (e.target as HTMLSelectElement).value || null
  priorityError.value = null
  try {
    await patchPriority(props.project, props.node.id, value)
    if (detail.value) {
      detail.value = {
        ...detail.value,
        frontmatter: { ...detail.value.frontmatter, priority: value ?? undefined },
      }
    }
    graphStore.updateNodePriority(props.node.id, value)
  } catch (err: unknown) {
    priorityError.value = err instanceof Error ? err.message : 'Failed to update priority'
  } finally {
    priorityEditing.value = false
  }
}

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
    showStatusPanel.value = false
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

function triggerStatusCheck() {
  showStatusPanel.value = true
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
  'in-qa': '#ede9fe',
}
const STATUS_TEXT: Record<string, string> = {
  done: '#065f46',
  approved: '#1e40af',
  'in-progress': '#92400e',
  blocked: '#991b1b',
  'in-qa': '#6d28d9',
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

            <!-- Labels badges -->
            <template v-if="node.labels && node.labels.length">
              <span
                v-for="lbl in node.labels"
                :key="lbl"
                class="meta-label-badge"
              >{{ lbl }}</span>
            </template>

            <!-- Priority display / inline edit -->
            <template v-if="!priorityEditing">
              <span
                v-if="detail?.frontmatter?.priority"
                class="meta-priority-badge"
                :style="{ background: PRIORITY_COLORS[detail.frontmatter.priority] + '33', color: PRIORITY_COLORS[detail.frontmatter.priority], borderColor: PRIORITY_COLORS[detail.frontmatter.priority] + '66' }"
                title="Click to edit priority"
                role="button"
                tabindex="0"
                @click="priorityEditing = true"
                @keydown.enter="priorityEditing = true"
              >{{ detail.frontmatter.priority }}</span>
              <span
                v-else-if="detail && !detail.frontmatter?.priority"
                class="meta-priority-none"
                title="Click to set priority"
                role="button"
                tabindex="0"
                @click="priorityEditing = true"
                @keydown.enter="priorityEditing = true"
              >No priority</span>
            </template>
            <select
              v-else
              class="priority-select"
              :value="detail?.frontmatter?.priority ?? ''"
              autofocus
              @change="onPriorityChange"
              @blur="priorityEditing = false"
            >
              <option value="">None</option>
              <option v-for="opt in PRIORITY_OPTIONS" :key="opt" :value="opt">
                {{ opt.charAt(0).toUpperCase() + opt.slice(1) }}
              </option>
            </select>
            <span v-if="priorityError" class="priority-error">{{ priorityError }}</span>
          </div>
          <h2 class="modal-title">{{ node.title || node.slug }}</h2>
          <div class="modal-path">{{ node.id }}</div>
        </div>

        <div class="modal-actions">
          <button class="action-btn action-btn--primary" @click="openEditor">Edit</button>
          <button class="action-btn" @click="showTransition = true">Change Status</button>
          <button class="action-btn" @click="showRunAgent = true">Run Agent</button>
          <button
            v-if="node.lineage"
            class="action-btn action-btn--check"
            :disabled="statusCheckLoading"
            @click="triggerStatusCheck"
          >
            <span v-if="statusCheckLoading" class="action-spinner" aria-hidden="true"></span>
            Check status
          </button>
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

        <StatusCheckPanel
          v-if="showStatusPanel && node.lineage"
          :project="project"
          :lineage="node.lineage"
          class="modal-status-panel"
          @close="showStatusPanel = false"
        />

        <ArtifactRunHistory
          v-if="node"
          :project="project"
          :target-path="node.id"
          @select-run="selectedRunId = $event"
        />

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

    <RunDetailModal
      v-if="selectedRunId"
      :project="project"
      :run-id="selectedRunId"
      @close="selectedRunId = null"
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
.meta-label-badge {
  display: inline-block;
  padding: 1px 7px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
  background: var(--color-surface);
}
.meta-priority-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 600;
  border: 1px solid transparent;
  cursor: pointer;
  text-transform: capitalize;
}
.meta-priority-badge:hover { opacity: 0.8; }
.meta-priority-none {
  font-size: 11px;
  color: var(--color-text-muted);
  cursor: pointer;
  opacity: 0.6;
}
.meta-priority-none:hover { opacity: 1; }
.priority-select {
  font-size: 11px;
  padding: 1px 4px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-accent);
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
}
.priority-error {
  font-size: 11px;
  color: #dc2626;
}
.action-btn--check {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
}
.action-btn--check:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.action-spinner {
  display: inline-block;
  width: 11px;
  height: 11px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: modal-spin 0.6s linear infinite;
}
@keyframes modal-spin {
  to { transform: rotate(360deg); }
}
@media (prefers-reduced-motion: reduce) {
  .action-spinner { animation: none; }
}
.modal-status-panel {
  margin: 0 var(--space-6) var(--space-4);
  flex-shrink: 0;
}
</style>
