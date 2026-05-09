<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()
</script>

<template>
  <Teleport to="body">
    <div class="toast-container" aria-live="polite" aria-atomic="false">
      <TransitionGroup name="toast">
        <div
          v-for="toast in ui.toasts"
          :key="toast.id"
          class="toast"
          :class="`toast--${toast.type}`"
          role="status"
        >
          <span class="toast-message">{{ toast.message }}</span>
          <button class="toast-close" aria-label="Dismiss" @click="ui.dismiss(toast.id)">
            ✕
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-container {
  position: fixed;
  bottom: var(--space-4);
  right: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  z-index: 9999;
  pointer-events: none;
}
.toast {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  font-size: var(--text-sm);
  max-width: 380px;
  pointer-events: all;
}
.toast--info  { background: var(--color-surface); border: 1px solid var(--color-border); color: var(--color-text); }
.toast--success { background: #d1fae5; border: 1px solid #6ee7b7; color: #065f46; }
.toast--error   { background: #fee2e2; border: 1px solid #fca5a5; color: #991b1b; }
.toast-message { flex: 1; }
.toast-close {
  background: none;
  border: none;
  cursor: pointer;
  font-size: var(--text-xs);
  opacity: 0.6;
  padding: 0;
  line-height: 1;
}
.toast-close:hover { opacity: 1; }
.toast-enter-active, .toast-leave-active { transition: all 0.2s; }
.toast-enter-from { opacity: 0; transform: translateY(8px); }
.toast-leave-to   { opacity: 0; transform: translateY(8px); }
</style>
