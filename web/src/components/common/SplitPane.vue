<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed } from 'vue'

const props = withDefaults(defineProps<{
  defaultRatio?: number
  minTopPx?: number
  minBottomPx?: number
}>(), {
  defaultRatio: 0.6,
  minTopPx: 80,
  minBottomPx: 48,
})

const emit = defineEmits<{
  (e: 'resize', ratio: number): void
}>()

const containerRef = ref<HTMLElement | null>(null)
const ratio = ref(props.defaultRatio)
const collapsed = ref(false)
const isDragging = ref(false)

const topStyle = computed(() => {
  if (collapsed.value) return { flex: '1 1 auto', minHeight: '0' }
  return { flex: `0 0 calc(${ratio.value * 100}% - 4px)`, minHeight: '0' }
})

const bottomStyle = computed(() => {
  if (collapsed.value) return { flex: '0 0 0px', overflow: 'hidden', minHeight: '0' }
  return { flex: '1 1 auto', minHeight: '0' }
})

function toggleCollapse() {
  collapsed.value = !collapsed.value
}

function startDrag(e: PointerEvent) {
  if (collapsed.value) return
  isDragging.value = true
  ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
}

function onDrag(e: PointerEvent) {
  if (!isDragging.value || !containerRef.value) return
  const rect = containerRef.value.getBoundingClientRect()
  const totalHeight = rect.height
  if (totalHeight === 0) return
  const relY = e.clientY - rect.top
  const minRatio = props.minTopPx / totalHeight
  const maxRatio = 1 - props.minBottomPx / totalHeight
  ratio.value = Math.max(minRatio, Math.min(maxRatio, relY / totalHeight))
}

function stopDrag(e: PointerEvent) {
  if (!isDragging.value) return
  isDragging.value = false
  ;(e.target as HTMLElement).releasePointerCapture(e.pointerId)
  emit('resize', ratio.value)
}

function onDividerKeyDown(e: KeyboardEvent) {
  if (e.key === 'ArrowUp') {
    ratio.value = Math.max(0.1, ratio.value - 0.02)
    if (collapsed.value) collapsed.value = false
    emit('resize', ratio.value)
    e.preventDefault()
  } else if (e.key === 'ArrowDown') {
    ratio.value = Math.min(0.9, ratio.value + 0.02)
    if (collapsed.value) collapsed.value = false
    emit('resize', ratio.value)
    e.preventDefault()
  } else if (e.key === 'Enter' || e.key === ' ') {
    toggleCollapse()
    e.preventDefault()
  }
}

function collapsePane() {
  collapsed.value = true
}

function expandPane() {
  collapsed.value = false
}

defineExpose({ collapsePane, expandPane, collapsed })
</script>

<template>
  <div ref="containerRef" class="split-pane">
    <div class="split-pane__top" :style="topStyle">
      <slot name="top" />
    </div>

    <div
      class="split-pane__divider"
      :class="{ 'split-pane__divider--dragging': isDragging }"
      tabindex="0"
      role="separator"
      aria-orientation="horizontal"
      :aria-valuenow="Math.round(ratio * 100)"
      :aria-label="collapsed ? 'Log pane collapsed — press Enter to expand' : 'Resize log pane — use arrow keys'"
      @pointerdown="startDrag"
      @pointermove="onDrag"
      @pointerup="stopDrag"
      @keydown="onDividerKeyDown"
    >
      <button
        class="split-pane__toggle"
        :aria-label="collapsed ? 'Expand log pane' : 'Collapse log pane'"
        tabindex="-1"
        @click.stop="toggleCollapse"
        @pointerdown.stop
      >
        <span class="split-pane__toggle-icon">{{ collapsed ? '▲' : '▼' }}</span>
      </button>
    </div>

    <div class="split-pane__bottom" :style="bottomStyle">
      <slot name="bottom" />
    </div>
  </div>
</template>

<style scoped>
.split-pane {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.split-pane__top {
  overflow: hidden;
}

.split-pane__bottom {
  overflow: hidden;
}

.split-pane__divider {
  height: 8px;
  flex: 0 0 8px;
  background: var(--color-border);
  cursor: row-resize;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  user-select: none;
  transition: background 0.15s;
}

.split-pane__divider:hover,
.split-pane__divider--dragging {
  background: var(--color-accent);
  opacity: 0.7;
}

.split-pane__divider:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: -2px;
}

.split-pane__toggle {
  position: absolute;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  width: 32px;
  height: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
  color: var(--color-text-muted);
  line-height: 1;
  z-index: 1;
}

.split-pane__toggle:hover {
  color: var(--color-text);
  border-color: var(--color-accent);
}

.split-pane__toggle-icon {
  font-size: 7px;
  line-height: 1;
  display: block;
  pointer-events: none;
}

@media (max-width: 768px) {
  .split-pane {
    flex-direction: column;
  }

  .split-pane__top {
    flex: 0 0 auto !important;
    max-height: 50vh;
    overflow-y: auto;
  }

  .split-pane__bottom {
    flex: 0 0 auto !important;
    min-height: 200px;
    max-height: 50vh;
    overflow-y: auto;
  }

  .split-pane__divider {
    cursor: default;
  }
}
</style>
