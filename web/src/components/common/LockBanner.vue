<script setup lang="ts">
import type { LockRow } from '@/types/api'

defineProps<{ lock: LockRow }>()
</script>

<template>
  <div class="lock-banner" role="alert">
    <span class="lock-icon">&#128274;</span>
    Locked by <strong>{{ lock.holder }}</strong>
    <span class="lock-kind">({{ lock.kind }})</span>
    &mdash; read-only while lock is held
  </div>
</template>

<style scoped>
.lock-banner {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-6);
  background: #fef3c7;
  color: #92400e;
  font-size: var(--text-sm);
  border-bottom: 1px solid #fde68a;
  flex-shrink: 0;
}
@media (prefers-color-scheme: dark) {
  .lock-banner { background: #451a03; color: #fbbf24; border-color: #78350f; }
}
.lock-icon { font-style: normal; }
.lock-kind { opacity: 0.7; }
</style>
