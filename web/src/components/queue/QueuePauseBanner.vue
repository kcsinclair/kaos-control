<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed } from 'vue'
import { useQueueStore } from '@/stores/queue'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useNow } from '@/composables/useNow'

const queueStore = useQueueStore()
const authStore = useAuthStore()
const ui = useUiStore()
const now = useNow()

// Product-owner or devops can resume manually.
const canResume = computed(() => {
  const roles = Object.values(authStore.me?.roles ?? {}).flat()
  return roles.includes('product-owner') || roles.includes('devops')
})

const pauseReasonLabel = computed(() => {
  const r = queueStore.snapshot.pause_reason
  if (r === 'rate_limit') return 'Rate limit reached'
  if (r === 'manual') return 'Manually paused'
  return 'Paused'
})

// Countdown to paused_until.
const countdownLabel = computed(() => {
  const until = queueStore.pausedUntilDate
  if (!until) return null
  const diffMs = until.getTime() - now.value.getTime()
  if (diffMs <= 0) return 'now'
  const diffSec = Math.ceil(diffMs / 1000)
  if (diffSec < 60) return `${diffSec}s`
  const mins = Math.floor(diffSec / 60)
  const secs = diffSec % 60
  if (mins < 60) return secs > 0 ? `${mins}m ${secs}s` : `${mins}m`
  const hrs = Math.floor(mins / 60)
  const rem = mins % 60
  return rem > 0 ? `${hrs}h ${rem}m` : `${hrs}h`
})

const resumesAtLabel = computed(() => {
  const until = queueStore.pausedUntilDate
  if (!until) return null
  return until.toLocaleString()
})

async function handleResume() {
  try {
    await queueStore.resume()
    ui.success('Queue resumed')
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to resume queue')
  }
}
</script>

<template>
  <div class="pause-banner" role="alert">
    <div class="pause-banner-content">
      <span class="pause-icon">⏸</span>
      <div class="pause-text">
        <strong class="pause-reason">{{ pauseReasonLabel }}</strong>
        <template v-if="resumesAtLabel">
          — resumes {{ resumesAtLabel }}
          <span v-if="countdownLabel" class="pause-countdown">(in {{ countdownLabel }})</span>
        </template>
      </div>
    </div>
    <button v-if="canResume" class="btn-resume" @click="handleResume">Resume now</button>
  </div>
</template>

<style scoped>
.pause-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding: var(--space-3) var(--space-6);
  background: #fffbeb;
  border: 1px solid #fcd34d;
  border-radius: var(--radius-md);
  margin-bottom: var(--space-4);
}
@media (prefers-color-scheme: dark) {
  .pause-banner { background: #422006; border-color: #92400e; color: #fcd34d; }
}
.pause-banner-content {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  font-size: var(--text-sm);
  color: #92400e;
}
@media (prefers-color-scheme: dark) {
  .pause-banner-content { color: #fcd34d; }
}
.pause-icon { font-size: 1.1em; }
.pause-reason { font-weight: 600; }
.pause-countdown {
  font-size: 11px;
  opacity: 0.8;
}
.btn-resume {
  padding: var(--space-1) var(--space-4);
  background: #92400e;
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
}
.btn-resume:hover { opacity: 0.88; }
</style>
