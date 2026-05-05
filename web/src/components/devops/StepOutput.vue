<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'

const props = defineProps<{
  lines: string[]
  failed?: boolean
}>()

const scrollEl = ref<HTMLPreElement | null>(null)

// Auto-scroll to bottom whenever lines grow
watch(
  () => props.lines.length,
  async () => {
    await nextTick()
    if (scrollEl.value) {
      scrollEl.value.scrollTop = scrollEl.value.scrollHeight
    }
  },
)
</script>

<template>
  <pre
    ref="scrollEl"
    class="step-output"
    :class="{ 'step-output--failed': props.failed }"
  >{{ props.lines.join('\n') }}</pre>
</template>

<style scoped>
.step-output {
  font-family: monospace;
  font-size: 11px;
  background: #0f172a;
  color: #e2e8f0;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  margin: var(--space-1) 0 var(--space-2);
  overflow-x: auto;
  overflow-y: auto;
  max-height: 240px;
  white-space: pre-wrap;
  word-break: break-all;
  line-height: 1.5;
  border: 1px solid transparent;
}
.step-output--failed {
  border-color: var(--color-error);
  background: #1f0a0a;
  color: #fca5a5;
}
</style>
