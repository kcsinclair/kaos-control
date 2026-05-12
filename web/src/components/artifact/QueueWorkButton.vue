<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { useQueueStore } from '@/stores/queue'
import { useAgentsStore } from '@/stores/agents'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { agentForArtifact } from '@/composables/useAgentForArtifact'
import type { ArtifactDetail } from '@/types/api'

const props = defineProps<{
  artifact: ArtifactDetail
  project: string
}>()

const queueStore = useQueueStore()
const agentsStore = useAgentsStore()
const authStore = useAuthStore()
const ui = useUiStore()

// Resolve which agent handles this artifact.
const agentName = computed(() =>
  agentForArtifact(props.artifact, agentsStore.agents),
)

// Check if the artifact is already in the queue (pending or running).
const queuedJob = computed(() => {
  const { running, pending } = queueStore.snapshot
  if (
    running &&
    running.project === props.project &&
    running.artifact_path === props.artifact.path
  ) {
    return running
  }
  return (
    pending.find(
      (j) =>
        j.project === props.project && j.artifact_path === props.artifact.path,
    ) ?? null
  )
})

// Whether the current user has a role that permits enqueueing this agent.
const userCanEnqueue = computed(() => {
  if (!agentName.value) return false
  const agent = agentsStore.agents.find((a) => a.name === agentName.value)
  if (!agent) return false
  const userRoles = authStore.rolesForProject(props.project)
  // product-owner is always allowed (mirrors backend logic)
  if (userRoles.includes('product-owner')) return true
  return agent.roles.some((r) => userRoles.includes(r))
})

// The button is visible when the artifact is approved and an agent is mapped.
const visible = computed(
  () => props.artifact.status === 'approved' && agentName.value !== null,
)

// Tooltip for the disabled state.
const disabledReason = computed(() => {
  if (!agentName.value) return 'No agent configured for this artifact type'
  if (!userCanEnqueue.value) return 'You do not have the required role to queue this agent'
  if (queuedJob.value) return null // queued badge shown instead
  return null
})

async function handleClick() {
  if (!agentName.value || !userCanEnqueue.value || queuedJob.value) return
  try {
    await queueStore.enqueue({
      project: props.project,
      artifact_path: props.artifact.path,
      agent: agentName.value,
    })
    ui.success('Added to queue')
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to queue')
  }
}
</script>

<template>
  <template v-if="visible">
    <!-- Already queued: show badge instead of button -->
    <span v-if="queuedJob" class="queued-badge">
      {{ queuedJob.state === 'running' ? 'Running…' : `Queued — position ${queuedJob.position}` }}
    </span>

    <!-- Not yet queued -->
    <button
      v-else
      class="btn-queue"
      :disabled="!userCanEnqueue"
      :title="disabledReason ?? `Queue for ${agentName}`"
      @click="handleClick"
    >
      Queue Work
    </button>
  </template>
</template>

<style scoped>
.btn-queue {
  padding: var(--space-1) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-queue:hover:not(:disabled) {
  background: var(--color-surface);
  color: var(--color-text);
}
.btn-queue:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.queued-badge {
  display: inline-flex;
  align-items: center;
  padding: var(--space-1) var(--space-3);
  border: 1px solid #f59e0b;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: #92400e;
  background: rgba(245, 158, 11, 0.1);
  white-space: nowrap;
}
</style>
