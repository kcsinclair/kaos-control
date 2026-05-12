<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAgentsStore } from '@/stores/agents'
import { useUiStore } from '@/stores/ui'
import * as artifactsApi from '@/api/artifacts'
import { typeToAgent } from '@/composables/useAgentForArtifact'
import type { AgentSummary, ArtifactRow } from '@/types/api'

const props = defineProps<{
  agent: AgentSummary
  project: string
}>()

const emit = defineEmits<{
  started: [runId: string]
  cancel: []
}>()

const agentsStore = useAgentsStore()
const ui = useUiStore()

const artifacts = ref<ArtifactRow[]>([])
const loading = ref(false)
const selectedPath = ref<string | null>(null)
const running = ref(false)

// Maps agent name to the artifact type it expects as input.
// Built by inverting the shared typeToAgent map; see composables/useAgentForArtifact.ts.
const agentInputTypeMap: Record<string, string> = Object.fromEntries(
  Object.entries(typeToAgent).map(([type, agent]) => [agent, type]),
)

const inputType = computed(() => agentInputTypeMap[props.agent.name])

const selectedArtifact = computed(
  () => artifacts.value.find((a) => a.path === selectedPath.value) ?? null,
)

async function fetchArtifacts() {
  loading.value = true
  try {
    const results: ArtifactRow[] = []

    // Fetch the primary input type artifacts (e.g., plan-backend for backend-developer).
    const filter: Record<string, string> = { status: 'approved' }
    if (inputType.value) filter.type = inputType.value
    const res = await artifactsApi.listArtifacts(props.project, filter)
    results.push(...(res.items ?? []))

    // For developer agents (plan-* input type), also include defects with
    // status "approved" that are assigned to this agent's role. QA creates
    // defects and assigns them to developer roles; those should appear here.
    if (inputType.value?.startsWith('plan-')) {
      const agentRoles = new Set(props.agent.roles)
      const defectRes = await artifactsApi.listArtifacts(props.project, {
        status: 'approved',
        type: 'defect',
      })
      const assignedDefects = (defectRes.items ?? []).filter((a) =>
        a.frontmatter.assignees?.some((assignee) => agentRoles.has(assignee.role)),
      )
      results.push(...assignedDefects)
    }

    artifacts.value = results
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to load artifacts')
  } finally {
    loading.value = false
  }
}

async function confirmRun() {
  if (!selectedArtifact.value) return
  running.value = true
  try {
    const runId = await agentsStore.startRun(
      props.project,
      props.agent.name,
      selectedArtifact.value.path,
      props.agent.roles[0],
    )
    ui.success(`Agent run started (${runId.slice(0, 8)}…)`)
    emit('started', runId)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to start run')
  } finally {
    running.value = false
  }
}

onMounted(fetchArtifacts)
</script>

<template>
  <div class="modal-overlay" @click.self="emit('cancel')">
    <div class="modal-panel" role="dialog" aria-modal="true" :aria-label="`Run ${agent.name}`">
      <div class="modal-header">
        <h3 class="modal-title">Run {{ agent.name }}</h3>
        <button class="btn-icon" aria-label="Cancel" @click="emit('cancel')">✕</button>
      </div>

      <div class="modal-body">
        <div v-if="loading" class="state-msg">Loading artifacts…</div>

        <template v-else-if="artifacts.length">
          <div class="artifact-list" role="listbox" :aria-label="`Artifacts for ${agent.name}`">
            <button
              v-for="artifact in artifacts"
              :key="artifact.path"
              class="artifact-item"
              :class="{ 'artifact-item--selected': selectedPath === artifact.path }"
              role="option"
              :aria-selected="selectedPath === artifact.path"
              @click="selectedPath = artifact.path"
            >
              <span class="artifact-title">{{ artifact.title }}</span>
              <span class="artifact-meta">
                <span class="artifact-lineage">{{ artifact.lineage }}</span>
                <span class="artifact-status">{{ artifact.status }}</span>
              </span>
              <span class="artifact-path">{{ artifact.path }}</span>
            </button>
          </div>
        </template>

        <div v-else class="state-msg">No eligible artifacts for this agent.</div>
      </div>

      <div class="modal-footer">
        <button
          class="btn-primary"
          :disabled="running || !selectedArtifact"
          @click="confirmRun"
        >
          {{ running ? 'Starting…' : 'Run' }}
        </button>
        <button class="btn-ghost" :disabled="running" @click="emit('cancel')">Cancel</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
}

.modal-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 520px;
  max-width: calc(100vw - 2rem);
  max-height: calc(100vh - 4rem);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.modal-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.btn-icon {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  padding: var(--space-1);
  border-radius: var(--radius-sm);
  line-height: 1;
}
.btn-icon:hover { background: var(--color-surface); color: var(--color-text); }

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-3) var(--space-4);
}

.state-msg {
  padding: var(--space-6) var(--space-4);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  text-align: center;
}

.artifact-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.artifact-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: none;
  text-align: left;
  cursor: pointer;
  font-family: inherit;
  transition: border-color 0.12s, background 0.12s;
}

.artifact-item:hover {
  background: var(--color-surface);
}

.artifact-item--selected {
  border-color: var(--color-accent);
  background: color-mix(in srgb, var(--color-accent) 10%, transparent);
}

.artifact-title {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}

.artifact-meta {
  display: flex;
  gap: var(--space-3);
  font-size: 11px;
  color: var(--color-text-muted);
}

.artifact-lineage {
  font-family: monospace;
}

.artifact-status {
  font-family: monospace;
}

.artifact-path {
  font-size: 11px;
  font-family: monospace;
  color: var(--color-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.modal-footer {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-4) var(--space-6);
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
}

.btn-primary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary:hover:not(:disabled) { opacity: 0.88; }

.btn-ghost {
  padding: var(--space-2) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-ghost:hover { background: var(--color-surface); }
.btn-ghost:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
