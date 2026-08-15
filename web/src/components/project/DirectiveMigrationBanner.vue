<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref } from 'vue'
import { FileText, Copy, Check } from 'lucide-vue-next'
import DirectiveMigrationModal from './DirectiveMigrationModal.vue'

const props = defineProps<{ project: string }>()

const showModal = ref(false)
const declined = ref(false)
const copied = ref(false)

const command = 'kaos-control migrate-directives'

function decline() {
  declined.value = true
}

async function copyCommand() {
  try {
    await navigator.clipboard.writeText(command)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    // clipboard not available; fail silently
  }
}

function onMigrated() {
  showModal.value = false
}
</script>

<template>
  <div class="directive-migration-banner" role="status">
    <FileText class="dmb-icon" />
    <div class="dmb-body">
      <span class="dmb-text">
        Directive handling is now multi-CLI — <code>AGENTS.md</code> is canonical, with
        <code>CLAUDE.md</code> and <code>GEMINI.md</code> pointing to it.
      </span>

      <div v-if="!declined" class="dmb-actions">
        <button class="dmb-btn dmb-btn--primary" @click="showModal = true">Migrate now</button>
        <button class="dmb-btn" @click="decline">Not now</button>
      </div>
      <div v-else class="dmb-cli-hint">
        <span>Run this later:</span>
        <code class="dmb-command">{{ command }}</code>
        <button class="dmb-copy-btn" :title="copied ? 'Copied!' : 'Copy command'" @click="copyCommand">
          <Check v-if="copied" :size="14" />
          <Copy v-else :size="14" />
        </button>
      </div>
    </div>

    <DirectiveMigrationModal
      v-if="showModal"
      :project="props.project"
      @close="showModal = false"
      @migrated="onMigrated"
    />
  </div>
</template>

<style scoped>
.directive-migration-banner {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-6);
  background: #eff6ff;
  color: #1e40af;
  border-bottom: 1px solid #bfdbfe;
  flex-shrink: 0;
}
@media (prefers-color-scheme: dark) {
  .directive-migration-banner { background: #172554; color: #93c5fd; border-color: #1e3a8a; }
}
.dmb-icon {
  width: 1.1rem;
  height: 1.1rem;
  flex-shrink: 0;
  margin-top: 2px;
}
.dmb-body {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
}
.dmb-text {
  font-size: var(--text-sm);
  line-height: 1.5;
}
.dmb-text code {
  font-family: monospace;
  font-size: 12px;
}
.dmb-actions {
  display: flex;
  gap: var(--space-2);
  flex-shrink: 0;
}
.dmb-btn {
  padding: var(--space-1) var(--space-3);
  border: 1px solid currentColor;
  border-radius: var(--radius-sm);
  background: transparent;
  color: inherit;
  font-size: var(--text-xs);
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
}
.dmb-btn--primary {
  background: currentColor;
}
.dmb-btn--primary {
  color: #fff;
  background: #1e40af;
  border-color: #1e40af;
}
@media (prefers-color-scheme: dark) {
  .dmb-btn--primary { background: #93c5fd; border-color: #93c5fd; color: #172554; }
}
.dmb-btn:hover { opacity: 0.85; }
.dmb-cli-hint {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  flex-wrap: wrap;
}
.dmb-command {
  font-family: monospace;
  font-size: 12px;
  padding: 2px var(--space-2);
  background: rgba(0, 0, 0, 0.06);
  border-radius: var(--radius-sm);
}
.dmb-copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-1);
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: inherit;
}
.dmb-copy-btn:hover { opacity: 0.85; }
</style>
