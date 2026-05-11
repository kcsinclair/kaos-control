<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { useAgentsStore } from '@/stores/agents'
import type { AgentSummary } from '@/types/api'

const props = defineProps<{
  agents: AgentSummary[]
}>()

const emit = defineEmits<{
  select: [agent: AgentSummary]
  edit: [agent: AgentSummary]
}>()

const route = useRoute()
const router = useRouter()
const agentsStore = useAgentsStore()

function isInline(agent: AgentSummary): boolean {
  return agent.driver === 'inline'
}

function isOllama(agent: AgentSummary): boolean {
  return agent.driver === 'ollama'
}

function driverLabel(agent: AgentSummary): string {
  if (agent.driver === 'ollama') return 'Ollama'
  if (agent.driver === 'claude-code-cli') return 'Claude Code'
  if (agent.driver === 'inline') return ''
  return agent.driver
}

function handleClick(agent: AgentSummary) {
  if (!isInline(agent)) {
    emit('select', agent)
  }
}

function handleBadgeClick(event: MouseEvent, agent: AgentSummary) {
  event.stopPropagation()
  if (!agent.active_status) return
  const project = route.params.project as string
  // The badge counts artifacts that are READY for this agent (status="approved"),
  // not artifacts already mid-run (status=active_status). Link to the same set
  // so clicking the badge surfaces what the launch dialog would show.
  const q = new URLSearchParams({ status: 'approved' })
  if (agent.source_types && agent.source_types.length > 0) {
    q.set('type', agent.source_types[0])
  }
  void router.push(`/p/${encodeURIComponent(project)}/artifacts?${q.toString()}`)
}

function readyCount(agent: AgentSummary): number {
  if (!agent.active_status) return 0
  return agentsStore.readyCounts[agent.name] ?? 0
}

function runningCount(agent: AgentSummary): number {
  return agentsStore.runningCountByAgent[agent.name] ?? 0
}
</script>

<template>
  <div v-if="props.agents.length" class="agent-panel-row">
    <button
      v-for="agent in props.agents"
      :key="agent.name"
      class="agent-panel"
      :class="{
        'agent-panel--disabled': isInline(agent),
        'agent-panel--running': runningCount(agent) > 0,
      }"
      :disabled="isInline(agent)"
      :aria-disabled="isInline(agent) ? 'true' : undefined"
      @click="handleClick(agent)"
    >
      <div class="panel-header">
        <span class="panel-name">{{ agent.name }}</span>
        <div class="panel-header-badges">
          <!-- Running count badge -->
          <span
            v-if="runningCount(agent) > 0"
            class="panel-running-badge"
            :aria-label="`${runningCount(agent)} run${runningCount(agent) === 1 ? '' : 's'} active`"
          >
            <span class="run-dot-small" />
            {{ runningCount(agent) }}
          </span>
          <button
            v-if="!isInline(agent)"
            class="panel-edit-btn"
            title="Edit agent"
            @click.stop="emit('edit', agent)"
          >✎</button>
        </div>
      </div>
      <span class="panel-roles">{{ agent.roles.join(', ') }}</span>
      <span v-if="!isInline(agent)" class="panel-driver" :data-driver="agent.driver">{{ driverLabel(agent) }}</span>
      <span v-if="agent.model" class="panel-model">{{ agent.model }}</span>
      <span v-if="isOllama(agent) && agent.ollama_instance" class="panel-model">{{ agent.ollama_instance }}</span>
      <span v-if="isInline(agent)" class="panel-inline-label">Externally driven</span>
      <!-- Ready-count badge (only for agents with active_status) -->
      <button
        v-if="agent.active_status"
        class="panel-ready-badge"
        :aria-label="`${readyCount(agent)} artifact${readyCount(agent) === 1 ? '' : 's'} ready`"
        @click="handleBadgeClick($event, agent)"
      >
        {{ readyCount(agent) }} ready
      </button>
    </button>
  </div>
</template>

<style scoped>
.agent-panel-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
}

.agent-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-3) var(--space-4);
  min-width: 140px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md, var(--radius-sm));
  background: var(--color-bg);
  text-align: left;
  font-family: inherit;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s, box-shadow 0.15s;
}

.agent-panel:hover:not(:disabled) {
  border-color: var(--color-accent);
  background: color-mix(in srgb, var(--color-accent) 8%, transparent);
}

.agent-panel:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.agent-panel--disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

/* Running state: green border + glow + pulse */
.agent-panel--running {
  border-color: #22c55e;
  box-shadow: 0 0 8px rgba(34, 197, 94, 0.3);
  animation: panel-pulse 1.5s ease-in-out infinite;
}

@keyframes panel-pulse {
  0%, 100% { box-shadow: 0 0 8px rgba(34, 197, 94, 0.3); }
  50%       { box-shadow: 0 0 14px rgba(34, 197, 94, 0.55); }
}

@media (prefers-reduced-motion: reduce) {
  .agent-panel--running {
    animation: none;
  }
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-1);
}
.panel-name {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  flex: 1;
}

.panel-header-badges {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}

.panel-running-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  border-radius: 9999px;
  padding: 2px 7px;
  font-size: 0.7rem;
  font-weight: 600;
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
  border: 1px solid rgba(34, 197, 94, 0.35);
  min-width: 2rem;
  white-space: nowrap;
}

.run-dot-small {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #22c55e;
  flex-shrink: 0;
  animation: panel-pulse-dot 1.5s ease-in-out infinite;
}

@keyframes panel-pulse-dot {
  0%, 100% { opacity: 1; }
  50%       { opacity: 0.4; }
}

@media (prefers-reduced-motion: reduce) {
  .run-dot-small {
    animation: none;
  }
}

.panel-edit-btn {
  background: none;
  border: none;
  padding: 2px 4px;
  font-size: 13px;
  color: var(--color-text-muted);
  cursor: pointer;
  border-radius: var(--radius-sm);
  opacity: 0;
  transition: opacity 0.15s;
}
.agent-panel:hover .panel-edit-btn {
  opacity: 1;
}
.panel-edit-btn:hover {
  background: var(--color-surface);
  color: var(--color-text);
}

.panel-roles {
  font-size: 11px;
  color: var(--color-text-muted);
}

.panel-model {
  font-size: 11px;
  color: var(--color-text-muted);
  font-family: monospace;
}

.panel-driver {
  display: inline-block;
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 99px;
  background: var(--color-border);
  color: var(--color-text-muted);
  align-self: flex-start;
}
.panel-driver[data-driver="ollama"] {
  background: #dbeafe;
  color: #1d4ed8;
}
.panel-driver[data-driver="claude-code-cli"] {
  background: #f3e8ff;
  color: #7e22ce;
}
.panel-inline-label {
  font-size: 10px;
  color: var(--color-text-muted);
  font-style: italic;
  margin-top: var(--space-1);
}

/* Ready-count badge */
.panel-ready-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  padding: 2px 8px;
  font-size: 0.7rem;
  font-weight: 600;
  background: color-mix(in srgb, var(--color-accent) 12%, transparent);
  color: var(--color-accent);
  border: 1px solid color-mix(in srgb, var(--color-accent) 35%, transparent);
  cursor: pointer;
  min-width: 4rem;
  align-self: flex-start;
  font-family: inherit;
  transition: background 0.15s;
}
.panel-ready-badge:hover {
  background: color-mix(in srgb, var(--color-accent) 22%, transparent);
}
</style>
