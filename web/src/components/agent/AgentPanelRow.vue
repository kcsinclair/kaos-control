<script setup lang="ts">
import type { AgentSummary } from '@/types/api'

const props = defineProps<{
  agents: AgentSummary[]
}>()

const emit = defineEmits<{
  select: [agent: AgentSummary]
}>()

function isInline(agent: AgentSummary): boolean {
  return agent.driver === 'inline'
}

function handleClick(agent: AgentSummary) {
  if (!isInline(agent)) {
    emit('select', agent)
  }
}
</script>

<template>
  <div v-if="props.agents.length" class="agent-panel-row">
    <button
      v-for="agent in props.agents"
      :key="agent.name"
      class="agent-panel"
      :class="{ 'agent-panel--disabled': isInline(agent) }"
      :disabled="isInline(agent)"
      :aria-disabled="isInline(agent) ? 'true' : undefined"
      @click="handleClick(agent)"
    >
      <span class="panel-name">{{ agent.name }}</span>
      <span class="panel-roles">{{ agent.roles.join(', ') }}</span>
      <span v-if="agent.model" class="panel-model">{{ agent.model }}</span>
      <span v-if="isInline(agent)" class="panel-inline-label">Externally driven</span>
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
  transition: border-color 0.15s, background 0.15s;
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

.panel-name {
  font-size: var(--text-sm);
  font-weight: 600;
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

.panel-inline-label {
  font-size: 10px;
  color: var(--color-text-muted);
  font-style: italic;
  margin-top: var(--space-1);
}
</style>
