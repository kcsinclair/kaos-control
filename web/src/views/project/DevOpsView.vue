<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useDevOpsStore } from '@/stores/devops'
import { useUiStore } from '@/stores/ui'
import { useWebSocket } from '@/composables/useWebSocket'
import type { WsEvent } from '@/types/api'
import type { RunHistoryEntry } from '@/stores/devops'
import PipelineCard from '@/components/devops/PipelineCard.vue'
import SplitPane from '@/components/common/SplitPane.vue'
import PipelineLogPane from '@/components/devops/PipelineLogPane.vue'

const route = useRoute()
const auth = useAuthStore()
const devops = useDevOpsStore()
const ui = useUiStore()

const project = route.params.project as string

const hasAccess = computed(() => {
  const roles = auth.rolesForProject(project)
  return roles.includes('product-owner') || roles.includes('devops')
})

const columnOrder = ['build', 'deploy', 'release']

const orderedTypes = computed((): string[] => {
  const types = Object.keys(devops.pipelinesByType)
  const known = columnOrder.filter((t) => types.includes(t))
  const dynamic = types.filter((t) => !columnOrder.includes(t)).sort()
  return [...known, ...dynamic]
})

// ── Log pane visibility ───────────────────────────────────────────────────────

const splitPaneRef = ref<InstanceType<typeof SplitPane> | null>(null)
const logPaneVisible = ref(false)

/** The pipeline name to display in the log pane header */
const logPipelineName = computed((): string | undefined => {
  const slug = devops.logPipelineSlug
  if (!slug) return undefined
  return devops.pipelines.find((p) => p.slug === slug)?.name ?? slug ?? undefined
})

function showLogPane() {
  logPaneVisible.value = true
  splitPaneRef.value?.expandPane()
}

function hideLogPane() {
  logPaneVisible.value = false
  splitPaneRef.value?.collapsePane()
}

function onLogPaneCollapse() {
  logPaneVisible.value = false
}

// Show the log pane whenever a run starts (live streaming)
watch(
  () => devops.logRunId,
  (id) => {
    if (id) showLogPane()
  },
)

// Handle user selecting a completed run from PipelineCard history
async function handleViewLog(entry: RunHistoryEntry) {
  try {
    await devops.loadRunLog(project, entry.runId, entry.pipelineSlug)
    showLogPane()
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to load run log.')
  }
}

// Clear log buffer on unmount (route change)
onUnmounted(() => {
  devops.clearLogBuffer()
})

onMounted(() => {
  if (hasAccess.value) {
    devops.fetchPipelines(project)
  }
})

useWebSocket(project, 'pipeline.run.started', (e: WsEvent) => {
  devops.handleRunStarted(e.payload)
})
useWebSocket(project, 'pipeline.step.started', (e: WsEvent) => {
  devops.handleStepStarted(e.payload)
})
useWebSocket(project, 'pipeline.step.output', (e: WsEvent) => {
  devops.handleStepOutput(e.payload)
})
useWebSocket(project, 'pipeline.step.completed', (e: WsEvent) => {
  devops.handleStepCompleted(e.payload)
})
useWebSocket(project, 'pipeline.run.completed', (e: WsEvent) => {
  devops.handleRunCompleted(e.payload)
})
</script>

<template>
  <div class="devops-view">
    <div v-if="!hasAccess" class="access-denied">
      <p class="access-denied__msg">You don't have permission to access DevOps pipelines.</p>
      <p class="access-denied__hint">Required role: <code>product-owner</code> or <code>devops</code>.</p>
    </div>

    <template v-else>
      <div class="devops-header">
        <h2 class="devops-title">DevOps Pipelines</h2>
      </div>

      <SplitPane
        ref="splitPaneRef"
        class="devops-split"
        :default-ratio="0.65"
        :min-top-px="120"
        :min-bottom-px="80"
        @resize="() => {}"
      >
        <!-- Top pane: pipeline kanban grid -->
        <template #top>
          <div class="devops-content">
            <div v-if="devops.loading" class="state-msg">Loading pipelines…</div>
            <div v-else-if="devops.loadError" class="state-msg state-msg--error">{{ devops.loadError }}</div>
            <div v-else-if="devops.pipelines.length === 0" class="state-msg">
              No pipelines found. Add YAML files to <code>lifecycle/devops/</code>.
            </div>
            <div v-else class="columns">
              <div
                v-for="type in orderedTypes"
                :key="type"
                class="column"
              >
                <h3 class="column-header">{{ type.charAt(0).toUpperCase() + type.slice(1) }}</h3>
                <div class="card-list">
                  <PipelineCard
                    v-for="pipeline in devops.pipelinesByType[type]"
                    :key="pipeline.slug"
                    :pipeline="pipeline"
                    :project="project"
                    @view-log="handleViewLog"
                  />
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- Bottom pane: log streaming -->
        <template #bottom>
          <PipelineLogPane
            v-if="logPaneVisible"
            :lines="devops.logBuffer"
            :run-completed="devops.logRunCompleted"
            :pipeline-name="logPipelineName"
            @collapse="onLogPaneCollapse"
          />
          <div v-else class="log-placeholder">
            <span class="log-placeholder__text">Log pane — run a pipeline or select a recent run to view logs.</span>
          </div>
        </template>
      </SplitPane>
    </template>
  </div>
</template>

<style scoped>
.devops-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.devops-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.devops-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.devops-split {
  flex: 1;
  min-height: 0;
}

.devops-content {
  height: 100%;
  overflow-y: auto;
  padding: var(--space-6);
}

.state-msg {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}

.state-msg--error {
  color: var(--color-error);
}

.columns {
  display: flex;
  gap: var(--space-6);
  align-items: flex-start;
}

.column {
  flex: 1;
  min-width: 220px;
  max-width: 380px;
}

.column-header {
  font-size: var(--text-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-text-muted);
  margin: 0 0 var(--space-3) 0;
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--color-border);
}

.card-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.log-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  background: #0f172a;
}

.log-placeholder__text {
  font-size: var(--text-xs);
  color: #475569;
  font-style: italic;
}

.access-denied {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: var(--space-2);
}

.access-denied__msg {
  font-size: var(--text-base);
  color: var(--color-text);
  margin: 0;
}

.access-denied__hint {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  margin: 0;
}
</style>
