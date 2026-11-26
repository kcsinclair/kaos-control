<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref } from 'vue'
import { useProviderSwitchStore } from '@/stores/providerSwitch'
import { useUiStore } from '@/stores/ui'
import type { ProviderTemplate } from '@/types/providerSwitch'

const props = defineProps<{
  project: string
  disabled?: boolean
}>()

const providerSwitchStore = useProviderSwitchStore()
const ui = useUiStore()

const open = ref(false)
const applying = ref<string | null>(null)

function toggle() {
  open.value = !open.value
}

async function applyTemplate(template: ProviderTemplate) {
  const agentNames = Object.keys(template.agents)
  const confirmed = window.confirm(
    `Apply template "${template.name}" to ${agentNames.length} agent(s)?\n\n${agentNames.join(', ')}`,
  )
  open.value = false
  if (!confirmed) return

  applying.value = template.name
  try {
    await providerSwitchStore.applyTemplate(props.project, template.name)
    ui.success(`Template "${template.name}" applied to ${agentNames.length} agent(s)`)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to apply provider template')
  } finally {
    applying.value = null
  }
}
</script>

<template>
  <div class="ptm">
    <button
      type="button"
      class="btn-secondary"
      :disabled="props.disabled || !providerSwitchStore.templates.length"
      @click="toggle"
    >
      Apply Preset Template
    </button>
    <div v-if="open" class="ptm-dropdown" role="menu">
      <button
        v-for="tpl in providerSwitchStore.templates"
        :key="tpl.name"
        type="button"
        class="ptm-item"
        role="menuitem"
        :disabled="applying === tpl.name"
        @click="applyTemplate(tpl)"
      >
        <span class="ptm-item-name">{{ tpl.name }}</span>
        <span v-if="tpl.description" class="ptm-item-desc">{{ tpl.description }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.ptm {
  position: relative;
  display: inline-block;
}
.ptm-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  min-width: 220px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  z-index: 20;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.ptm-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-2) var(--space-3);
  background: none;
  border: none;
  text-align: left;
  cursor: pointer;
  font-family: inherit;
}
.ptm-item:hover:not(:disabled) { background: var(--color-surface); }
.ptm-item:disabled { opacity: 0.5; cursor: not-allowed; }
.ptm-item-name { font-size: var(--text-sm); font-weight: 500; color: var(--color-text); }
.ptm-item-desc { font-size: 11px; color: var(--color-text-muted); }
.btn-secondary {
  padding: var(--space-2) var(--space-4);
  background: transparent;
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-secondary:hover:not(:disabled) { border-color: var(--color-text-muted); color: var(--color-text); }
.btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
