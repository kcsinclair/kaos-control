<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref } from 'vue'
import { FolderPlus, Copy, Check } from 'lucide-vue-next'

const props = defineProps<{ path: string }>()

const command = `kaos-control init ${props.path}`
const copied = ref(false)

async function copyCommand() {
  try {
    await navigator.clipboard.writeText(command)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    // clipboard not available; fail silently
  }
}
</script>

<template>
  <div class="init-required">
    <div class="init-required-card">
      <FolderPlus class="init-icon" />
      <h2 class="init-heading">Project not initialised</h2>
      <p class="init-body">
        Run the command below to scaffold the lifecycle directory structure for this project.
      </p>
      <div class="init-command-block">
        <code class="init-command">{{ command }}</code>
        <button class="init-copy-btn" :title="copied ? 'Copied!' : 'Copy command'" @click="copyCommand">
          <Check v-if="copied" class="copy-icon" />
          <Copy v-else class="copy-icon" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.init-required {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: var(--space-8);
}

.init-required-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
  max-width: 480px;
  width: 100%;
  padding: var(--space-8);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  text-align: center;
}

.init-icon {
  width: 2.5rem;
  height: 2.5rem;
  color: var(--color-accent);
  flex-shrink: 0;
}

.init-heading {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--color-text);
}

.init-body {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  line-height: 1.5;
}

.init-command-block {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4);
  font-family: monospace;
}

.init-command {
  flex: 1;
  font-size: var(--text-sm);
  color: var(--color-text);
  white-space: pre;
  overflow-x: auto;
  text-align: left;
}

.init-copy-btn {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-1);
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--color-text-muted);
  transition: color 0.15s;
}

.init-copy-btn:hover {
  color: var(--color-text);
}

.copy-icon {
  width: 1rem;
  height: 1rem;
}
</style>
