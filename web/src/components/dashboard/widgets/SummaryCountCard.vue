<script setup lang="ts">
import type { Component } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { useRouter } from 'vue-router'
import { computed } from 'vue'

const props = defineProps<{
  label: string
  value: number | string
  icon?: Component
  to?: RouteLocationRaw | null
}>()

const router = useRouter()
const isInteractive = computed(() => props.to != null)

const ariaLabel = computed(() =>
  isInteractive.value
    ? `View ${props.value} ${props.label.toLowerCase()} artifacts`
    : `${props.label}: ${props.value}`
)

function navigate() {
  if (props.to) router.push(props.to)
}
</script>

<template>
  <div
    class="summary-card"
    :class="{ 'summary-card--interactive': isInteractive }"
    :role="isInteractive ? 'link' : 'figure'"
    :aria-label="ariaLabel"
    :tabindex="isInteractive ? 0 : undefined"
    @click="isInteractive && navigate()"
    @keydown.enter="isInteractive && navigate()"
    @keydown.space.prevent="isInteractive && navigate()"
  >
    <div class="summary-card-icon" aria-hidden="true">
      <component :is="icon" v-if="icon" :size="20" />
    </div>
    <div class="summary-card-body">
      <span class="summary-card-value">{{ value }}</span>
      <span class="summary-card-label">{{ label }}</span>
    </div>
  </div>
</template>

<style scoped>
.summary-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
  outline: none;
}

.summary-card:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.summary-card-icon {
  color: var(--color-primary);
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.summary-card-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.summary-card-value {
  font-size: var(--text-2xl, 1.5rem);
  font-weight: 700;
  color: var(--color-text);
  line-height: 1;
}

.summary-card-label {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  margin-top: var(--space-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.summary-card--interactive {
  cursor: pointer;
  transition: box-shadow 0.15s ease, background-color 0.15s ease;
}

.summary-card--interactive:hover {
  background: var(--color-surface-elevated, color-mix(in srgb, var(--color-surface) 85%, var(--color-text) 15%));
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
}

.summary-card--interactive:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
</style>
