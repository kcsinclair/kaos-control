<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  label: string
  disabled?: boolean
}>()

const visible = ref(false)
const style = ref<{ top: string; left: string }>({ top: '0px', left: '0px' })

function show(event: MouseEvent | FocusEvent) {
  if (props.disabled) return
  const el = (event.currentTarget as HTMLElement).querySelector<HTMLElement>('.tooltip-anchor') ?? (event.currentTarget as HTMLElement)
  const rect = el.getBoundingClientRect()
  style.value = {
    top: `${rect.top + rect.height / 2}px`,
    left: `${rect.right + 8}px`,
  }
  visible.value = true
}

function hide() {
  visible.value = false
}
</script>

<template>
  <div
    class="tooltip-wrapper"
    @mouseenter="show"
    @mouseleave="hide"
    @focusin="show"
    @focusout="hide"
  >
    <slot />
    <Teleport to="body">
      <div
        v-if="visible && !disabled"
        class="sidebar-tooltip"
        :style="style"
        role="tooltip"
        aria-hidden="true"
      >
        {{ label }}
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.tooltip-wrapper {
  display: contents;
}
</style>

<style>
.sidebar-tooltip {
  position: fixed;
  z-index: 9999;
  transform: translateY(-50%);
  padding: 4px 10px;
  border-radius: 5px;
  background: #0f172a;
  color: #e2e8f0;
  font-size: 0.8125rem;
  font-weight: 500;
  white-space: nowrap;
  border: 1px solid #334155;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
  pointer-events: none;
  line-height: 1.4;
}
</style>
