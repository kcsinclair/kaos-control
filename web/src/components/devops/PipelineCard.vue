<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Pipeline } from '@/api/devops'
import { useDevOpsStore } from '@/stores/devops'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/api/client'
import StepProgress from '@/components/devops/StepProgress.vue'
import StepOutput from '@/components/devops/StepOutput.vue'
import RunHistory from '@/components/devops/RunHistory.vue'
import { Pencil, CheckCircle, XCircle, MinusCircle } from 'lucide-vue-next'
import { useNow } from '@/composables/useNow'
import { formatRelativeTime } from '@/composables/useRunFormatters'

const props = defineProps<{
  pipeline: Pipeline
  project: string
}>()

const emit = defineEmits<{
  (e: 'edit', slug: string): void
}>()

const devops = useDevOpsStore()
const ui = useUiStore()
const auth = useAuthStore()

const canEdit = computed(() => {
  const roles = auth.rolesForProject(props.project)
  return roles.includes('product-owner') || roles.includes('devops')
})

const running = ref(false)
const cancelling = ref(false)
// Track which step indices have their output pane open
const outputOpen = ref(new Set<number>())

const activeRun = computed(() => devops.activeRuns.get(props.pipeline.slug))
const isActive = computed(() => activeRun.value?.overallStatus === 'running')
const showSteps = computed(() => activeRun.value != null)

const now = useNow()
const latestRun = computed(() => devops.latestRunForPipeline(props.pipeline.slug))

function toggleOutput(index: number) {
  if (outputOpen.value.has(index)) {
    outputOpen.value.delete(index)
  } else {
    outputOpen.value.add(index)
  }
  // Trigger reactivity by reassigning
  outputOpen.value = new Set(outputOpen.value)
}

async function handleRun() {
  running.value = true
  outputOpen.value = new Set()
  try {
    await devops.runPipeline(props.project, props.pipeline.slug)
  } catch (e: unknown) {
    if (e instanceof ApiError && e.status === 409) {
      ui.error('Pipeline is already running.')
    } else if (e instanceof ApiError && e.status === 403) {
      ui.error('You do not have permission to run this pipeline.')
    } else {
      ui.error(e instanceof Error ? e.message : 'Failed to start pipeline.')
    }
  } finally {
    running.value = false
  }
}

async function handleCancel() {
  cancelling.value = true
  try {
    await devops.cancelPipeline(props.project, props.pipeline.slug)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to cancel pipeline.')
  } finally {
    cancelling.value = false
  }
}
</script>

<template>
  <div
    class="pipeline-card"
    :class="{
      'pipeline-card--running': isActive,
      'pipeline-card--passed': activeRun?.overallStatus === 'passed',
      'pipeline-card--failed': activeRun?.overallStatus === 'failed',
    }"
  >
    <div class="card-header">
      <span class="pipeline-name">{{ props.pipeline.name }}</span>
      <span class="type-badge">{{ props.pipeline.type }}</span>
    </div>
    <div class="card-meta">
      <span class="step-count">{{ props.pipeline.steps.length }} step{{ props.pipeline.steps.length !== 1 ? 's' : '' }}</span>
      <span v-if="activeRun?.overallStatus === 'passed'" class="run-status run-status--passed">Passed</span>
      <span v-else-if="activeRun?.overallStatus === 'failed'" class="run-status run-status--failed">Failed</span>
      <span v-else-if="activeRun?.overallStatus === 'cancelled'" class="run-status run-status--cancelled">Cancelled</span>
      <span v-else-if="isActive" class="run-status run-status--running">Running</span>
      <!-- Latest historical run badge when no active run is displayed -->
      <span
        v-else-if="latestRun"
        class="latest-run-badge"
        :class="`latest-run-badge--${latestRun.status}`"
        :title="`Last run: ${latestRun.status} — ${new Date(latestRun.started_at).toLocaleString()}`"
      >
        <CheckCircle v-if="latestRun.status === 'passed'" :size="11" />
        <XCircle v-else-if="latestRun.status === 'failed'" :size="11" />
        <MinusCircle v-else :size="11" />
        {{ formatRelativeTime(latestRun.started_at, now) }}
      </span>
    </div>

    <!-- Step list shown when a run is active or completed -->
    <div v-if="showSteps && activeRun" class="step-list">
      <template v-for="(step, i) in activeRun.steps" :key="i">
        <StepProgress
          :step="step"
          :index="i"
          :output-open="outputOpen.has(i)"
          @toggle-output="toggleOutput(i)"
        />
        <StepOutput
          v-if="outputOpen.has(i)"
          :lines="step.output"
          :failed="step.status === 'failed'"
        />
      </template>
    </div>

    <RunHistory
      :pipeline-slug="props.pipeline.slug"
      :project="props.project"
    />

    <div class="card-actions">
      <button
        v-if="!isActive"
        class="btn-run"
        :disabled="running"
        @click="handleRun"
      >{{ running ? 'Starting…' : 'Run' }}</button>
      <button
        v-else
        class="btn-cancel"
        :disabled="cancelling"
        @click="handleCancel"
      >{{ cancelling ? 'Cancelling…' : 'Cancel' }}</button>
      <button
        v-if="canEdit"
        class="btn-edit"
        :disabled="devops.anyRunning"
        :title="devops.anyRunning ? 'Editing is disabled while a pipeline is running' : 'Edit pipeline'"
        @click="emit('edit', props.pipeline.slug)"
      >
        <Pencil :size="14" />
      </button>
    </div>
  </div>

</template>

<style scoped>
.pipeline-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  transition: border-color 0.2s;
}
.pipeline-card--running {
  border-color: var(--color-accent);
}
.pipeline-card--passed {
  border-color: #22c55e;
}
.pipeline-card--failed {
  border-color: var(--color-error);
}
.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-2);
}
.pipeline-name {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  flex: 1;
  min-width: 0;
  word-break: break-word;
}
.type-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--color-border);
  color: var(--color-text-muted);
  flex-shrink: 0;
}
.card-meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.step-count {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
}
.run-status {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 99px;
}
.run-status--running {
  background: var(--badge-approved-bg);
  color: var(--badge-approved-text);
}
.run-status--passed {
  background: var(--badge-done-bg);
  color: var(--badge-done-text);
}
.run-status--failed {
  background: var(--badge-blocked-bg);
  color: var(--badge-blocked-text);
}
.run-status--cancelled {
  background: var(--color-border);
  color: var(--color-text-muted);
}
.latest-run-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 10px;
  font-weight: 500;
}
.latest-run-badge--passed {
  color: #22c55e;
}
.latest-run-badge--failed {
  color: var(--color-error);
}
.latest-run-badge--cancelled {
  color: var(--color-text-muted);
}
.step-list {
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-2);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.card-actions {
  display: flex;
  gap: var(--space-2);
}
.btn-run {
  padding: var(--space-1) var(--space-3);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-run:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.btn-run:not(:disabled):hover { opacity: 0.88; }
.btn-cancel {
  padding: var(--space-1) var(--space-3);
  background: transparent;
  color: var(--color-error);
  border: 1px solid var(--color-error);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-cancel:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.btn-cancel:not(:disabled):hover {
  background: var(--badge-blocked-bg);
}
.btn-edit {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-1) var(--space-2);
  background: transparent;
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  margin-left: auto;
}
.btn-edit:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.btn-edit:not(:disabled):hover {
  background: var(--color-surface);
  color: var(--color-text);
}
</style>
