<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useAgentsStore } from '@/stores/agents'
import { useQueueStore } from '@/stores/queue'
import { useUiStore } from '@/stores/ui'
import { agentForArtifact } from '@/composables/useAgentForArtifact'
import { transitionArtifact } from '@/api/artifacts'
import type { ArtifactDetail } from '@/types/api'

const props = defineProps<{
  project: string
  path: string
  artifact: ArtifactDetail
}>()

const emit = defineEmits<{
  dismiss: []
  approved: []
  requeued: []
}>()

const agentsStore = useAgentsStore()
const queueStore = useQueueStore()
const ui = useUiStore()

// Developer-raised case: the blocked artefact is itself a plan-* artefact, so
// the agent that would resume work on it is the one that raised the
// questions in the first place (Resolved Q5 — "the role which actions it").
const requeueAgent = computed(() => agentForArtifact(props.artifact, agentsStore.agents))
const isDeveloperRaised = computed(
  () => props.artifact.type.startsWith('plan-') && !!requeueAgent.value,
)

const approving = ref(false)
const requeuing = ref(false)

async function approveAndContinue() {
  approving.value = true
  try {
    await transitionArtifact(props.project, props.path, 'approved')
    ui.success('Approved — handed off to the planning-analyst')
    emit('approved')
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to approve')
  } finally {
    approving.value = false
  }
}

async function requeueRole() {
  if (!requeueAgent.value) return
  requeuing.value = true
  try {
    await queueStore.enqueue({
      project: props.project,
      artifact_path: props.path,
      agent: requeueAgent.value,
    })
    ui.success(`Requeued for ${requeueAgent.value}`)
    emit('requeued')
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to requeue')
  } finally {
    requeuing.value = false
  }
}
</script>

<template>
  <div class="prp-banner">
    <span class="prp-text">Answers saved — what's next?</span>
    <button
      v-if="isDeveloperRaised"
      class="btn-primary"
      :disabled="requeuing"
      @click="requeueRole"
    >{{ requeuing ? 'Requeuing…' : `Requeue ${requeueAgent}` }}</button>
    <button
      v-else
      class="btn-primary"
      :disabled="approving"
      @click="approveAndContinue"
    >{{ approving ? 'Approving…' : 'Approve & continue' }}</button>
    <button class="btn-ghost" @click="emit('dismiss')">Dismiss</button>
  </div>
</template>

<style scoped>
.prp-banner {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
  padding: var(--space-3) var(--space-6);
  border-bottom: 1px solid #bfdbfe;
  background: #eff6ff;
  color: #1d4ed8;
  font-size: var(--text-sm);
  flex-shrink: 0;
}
@media (prefers-color-scheme: dark) {
  .prp-banner { background: #1e3a5f; color: #93c5fd; border-color: #1e40af; }
}
.prp-text { font-weight: 500; }
.btn-primary {
  margin-left: auto;
  padding: var(--space-1) var(--space-4);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-primary:hover:not(:disabled) { opacity: 0.88; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-ghost {
  padding: var(--space-1) var(--space-3);
  background: none;
  border: 1px solid currentColor;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: inherit;
  cursor: pointer;
}
.btn-ghost:hover { opacity: 0.8; }
</style>
