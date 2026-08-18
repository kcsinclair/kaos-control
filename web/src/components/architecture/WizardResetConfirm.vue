<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
// Defect: arch-wizard-no-reset-button. Confirms the destructive "Start
// Again" action before the wizard shell discards all in-progress
// selections and answers and returns to the first step.
const emit = defineEmits<{
  confirm: []
  close: []
}>()
</script>

<template>
  <Teleport to="body">
    <div
      class="modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="reset-wizard-title"
      @keydown.escape="emit('close')"
    >
      <div class="modal-panel">
        <div class="modal-header">
          <h2 id="reset-wizard-title" class="modal-title">Start Again</h2>
          <button class="modal-close" aria-label="Close" @click="emit('close')">✕</button>
        </div>

        <div class="modal-body">
          <p class="confirm-message">
            This discards all selections and answers made so far in the Architecture Wizard and
            returns you to the first step.
          </p>
        </div>

        <div class="modal-footer">
          <button class="btn-secondary" @click="emit('close')">Cancel</button>
          <button class="btn-danger" @click="emit('confirm')">Start Again</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  padding: var(--space-6);
}
.modal-panel {
  position: relative;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 100%;
  max-width: 440px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-5) var(--space-6) var(--space-4);
  border-bottom: 1px solid var(--color-border);
}
.modal-title {
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
  margin: 0;
}
.modal-close {
  background: none;
  border: none;
  font-size: var(--text-lg);
  color: var(--color-text-muted);
  cursor: pointer;
  line-height: 1;
  padding: var(--space-1);
}
.modal-close:hover { color: var(--color-text); }
.modal-body {
  padding: var(--space-5) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.confirm-message {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text);
  line-height: 1.5;
}
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-6);
  border-top: 1px solid var(--color-border);
}
.btn-secondary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-secondary:hover { background: var(--color-border); }
.btn-danger {
  padding: var(--space-2) var(--space-5);
  background: #dc2626;
  color: #fff;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-danger:hover { background: #b91c1c; }
</style>
