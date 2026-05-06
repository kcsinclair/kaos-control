<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import type { LogLine } from '@/stores/devops'
import { useVirtualScroll, VIRTUAL_SCROLL_ROW_HEIGHT } from '@/composables/useVirtualScroll'

// ── Virtual scrolling threshold ───────────────────────────────────────────────
const VIRTUAL_THRESHOLD = 10_000

const props = defineProps<{
  lines: LogLine[]
  runCompleted: boolean
  pipelineName?: string
}>()

const emit = defineEmits<{
  (e: 'collapse'): void
}>()

// ── Step filter (Milestone 3 hook: selectedStep is declared here, used by filter) ──
const selectedStep = ref<string>('__all__')

const availableSteps = computed((): string[] => {
  const seen = new Set<string>()
  for (const line of props.lines) {
    if (line.stepName) seen.add(line.stepName)
  }
  return Array.from(seen)
})

const filteredLines = computed((): LogLine[] => {
  if (selectedStep.value === '__all__') return props.lines
  return props.lines.filter(
    (l) => l.stepName === selectedStep.value || l.kind === 'run-start' || l.kind === 'run-end',
  )
})

// ── Virtual scrolling ─────────────────────────────────────────────────────────
const scrollEl = ref<HTMLElement | null>(null)
const useVirtual = computed(() => filteredLines.value.length > VIRTUAL_THRESHOLD)

const { totalHeight, visibleItems, handleScroll, ROW_HEIGHT } = useVirtualScroll(filteredLines, scrollEl)

// ── Auto-follow ────────────────────────────────────────────────────────────────
const autoFollow = ref(true)
const atBottom = ref(true)

function onScroll(e: Event) {
  handleScroll(e)
  const el = e.currentTarget as HTMLElement
  // Detect whether the user has scrolled away from the bottom
  const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
  const wasAtBottom = atBottom.value
  atBottom.value = distFromBottom < 8
  if (wasAtBottom && !atBottom.value) {
    // User scrolled up — pause auto-follow
    autoFollow.value = false
  } else if (!wasAtBottom && atBottom.value) {
    // User scrolled back to bottom — re-engage auto-follow
    autoFollow.value = true
  }
}

function scrollToBottom() {
  if (!scrollEl.value) return
  scrollEl.value.scrollTop = scrollEl.value.scrollHeight
}

function engageFollow() {
  autoFollow.value = true
  nextTick(scrollToBottom)
}

// When auto-follow is active and new lines arrive, scroll to bottom
watch(
  () => filteredLines.value.length,
  () => {
    if (autoFollow.value && !props.runCompleted) {
      nextTick(scrollToBottom)
    }
  },
)

// When run completes, disable auto-follow
watch(
  () => props.runCompleted,
  (done) => {
    if (done) autoFollow.value = false
  },
)

// Scroll to bottom on mount if auto-follow is on
onMounted(() => {
  if (autoFollow.value) nextTick(scrollToBottom)
})

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatTs(ts: number): string {
  return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function formatDuration(ms: number | undefined): string {
  if (ms == null) return ''
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

// ── Keyboard navigation ────────────────────────────────────────────────────────
function onPaneKeyDown(e: KeyboardEvent) {
  if (!scrollEl.value) return
  switch (e.key) {
    case 'ArrowDown':
      scrollEl.value.scrollTop += ROW_HEIGHT * 3
      e.preventDefault()
      break
    case 'ArrowUp':
      scrollEl.value.scrollTop -= ROW_HEIGHT * 3
      e.preventDefault()
      break
    case 'PageDown':
      scrollEl.value.scrollTop += scrollEl.value.clientHeight
      e.preventDefault()
      break
    case 'PageUp':
      scrollEl.value.scrollTop -= scrollEl.value.clientHeight
      e.preventDefault()
      break
    case 'End':
      scrollToBottom()
      e.preventDefault()
      break
    case 'Home':
      scrollEl.value.scrollTop = 0
      e.preventDefault()
      break
    case 'Escape':
      emit('collapse')
      e.preventDefault()
      break
  }
}
</script>

<template>
  <div class="log-pane" tabindex="0" @keydown="onPaneKeyDown">
    <!-- Header -->
    <div class="log-pane__header">
      <span class="log-pane__title">{{ props.pipelineName ? `Logs — ${props.pipelineName}` : 'Pipeline Logs' }}</span>

      <!-- Step filter (Milestone 3) -->
      <select
        v-if="availableSteps.length > 0"
        v-model="selectedStep"
        class="log-pane__step-filter"
        aria-label="Filter by step"
      >
        <option value="__all__">All steps</option>
        <option v-for="step in availableSteps" :key="step" :value="step">{{ step }}</option>
      </select>

      <!-- Follow button -->
      <button
        v-if="!autoFollow && !props.runCompleted"
        class="log-pane__follow-btn"
        @click="engageFollow"
      >
        Follow ↓
      </button>

      <span v-if="props.lines.length === 0" class="log-pane__hint">Waiting for output…</span>
    </div>

    <!-- Scroll container -->
    <div
      ref="scrollEl"
      class="log-pane__scroll"
      @scroll="onScroll"
    >
      <!-- Virtual scroll mode: only when line count exceeds threshold -->
      <template v-if="useVirtual">
        <div class="log-pane__spacer" :style="{ height: totalHeight + 'px' }">
          <div
            v-for="row in visibleItems"
            :key="row.index"
            class="log-row"
            :class="`log-row--${row.item.kind}`"
            :style="{ top: row.offsetTop + 'px' }"
          >
            <template v-if="row.item.kind === 'step-start'">
              <span class="log-row__step-label">{{ row.item.text }}</span>
              <span class="log-row__ts">{{ formatTs(row.item.timestamp) }}</span>
            </template>
            <template v-else-if="row.item.kind === 'step-end'">
              <span class="log-row__step-label">
                {{ row.item.text }} —
                <span :class="row.item.status === 'passed' ? 'log-row__ok' : 'log-row__fail'">{{ row.item.status }}</span>
                <span v-if="row.item.durationMs != null" class="log-row__dur">{{ formatDuration(row.item.durationMs) }}</span>
              </span>
            </template>
            <template v-else-if="row.item.kind === 'run-end'">
              <span class="log-row__terminal">
                Run
                <span :class="row.item.status === 'passed' ? 'log-row__ok' : 'log-row__fail'">{{ row.item.status }}</span>
                <span v-if="row.item.durationMs != null" class="log-row__dur">{{ formatDuration(row.item.durationMs) }}</span>
              </span>
            </template>
            <template v-else>
              <span class="log-row__text">{{ row.item.text }}</span>
            </template>
          </div>
        </div>
      </template>

      <!-- Normal mode: render all lines -->
      <template v-else>
        <div
          v-for="(line, i) in filteredLines"
          :key="i"
          class="log-row log-row--normal"
          :class="`log-row--${line.kind}`"
        >
          <template v-if="line.kind === 'step-start'">
            <span class="log-row__step-label">{{ line.text }}</span>
            <span class="log-row__ts">{{ formatTs(line.timestamp) }}</span>
          </template>
          <template v-else-if="line.kind === 'step-end'">
            <span class="log-row__step-label">
              {{ line.text }} —
              <span :class="line.status === 'passed' ? 'log-row__ok' : 'log-row__fail'">{{ line.status }}</span>
              <span v-if="line.durationMs != null" class="log-row__dur"> {{ formatDuration(line.durationMs) }}</span>
            </span>
          </template>
          <template v-else-if="line.kind === 'run-end'">
            <span class="log-row__terminal">
              Run
              <span :class="line.status === 'passed' ? 'log-row__ok' : 'log-row__fail'">{{ line.status }}</span>
              <span v-if="line.durationMs != null" class="log-row__dur"> {{ formatDuration(line.durationMs) }}</span>
            </span>
          </template>
          <template v-else>
            <span class="log-row__text">{{ line.text }}</span>
          </template>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.log-pane {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #0f172a;
  color: #e2e8f0;
  overflow: hidden;
  outline: none;
}

.log-pane:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: -2px;
}

/* ── Header ────────────────────────────────────────────────────────────────── */
.log-pane__header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: 4px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  flex-shrink: 0;
  background: #0f172a;
}

.log-pane__title {
  font-size: 11px;
  font-weight: 600;
  color: #94a3b8;
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.log-pane__hint {
  font-size: 11px;
  color: #475569;
  font-style: italic;
}

.log-pane__step-filter {
  font-size: 11px;
  background: #1e293b;
  color: #cbd5e1;
  border: 1px solid #334155;
  border-radius: 3px;
  padding: 2px 6px;
  cursor: pointer;
  max-width: 160px;
}

.log-pane__follow-btn {
  font-size: 11px;
  background: #1e3a5f;
  color: #93c5fd;
  border: 1px solid #2563eb;
  border-radius: 3px;
  padding: 2px 8px;
  cursor: pointer;
  flex-shrink: 0;
  white-space: nowrap;
}

.log-pane__follow-btn:hover {
  background: #1d4ed8;
  color: #fff;
}

/* ── Scroll area ────────────────────────────────────────────────────────────── */
.log-pane__scroll {
  flex: 1;
  overflow-y: auto;
  overflow-x: auto;
  padding: 4px 0;
  scroll-behavior: auto;
}

/* Spacer for virtual mode — items are absolutely positioned within */
.log-pane__spacer {
  position: relative;
  width: 100%;
}

/* ── Log rows ────────────────────────────────────────────────────────────────── */
.log-row {
  position: absolute;
  left: 0;
  right: 0;
  height: v-bind('VIRTUAL_SCROLL_ROW_HEIGHT + "px"');
  line-height: v-bind('VIRTUAL_SCROLL_ROW_HEIGHT + "px"');
  padding: 0 12px;
  font-family: monospace;
  font-size: 12px;
  white-space: pre;
  box-sizing: border-box;
  overflow: hidden;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Normal (non-virtual) rows are flow-positioned */
.log-row--normal {
  position: static;
  height: auto;
  min-height: v-bind('VIRTUAL_SCROLL_ROW_HEIGHT + "px"');
  line-height: v-bind('VIRTUAL_SCROLL_ROW_HEIGHT + "px"');
}

/* Output line */
.log-row--output {
  color: #e2e8f0;
}

/* Step boundary — start */
.log-row--step-start {
  background: #1e3a5f;
  color: #93c5fd;
  font-weight: 600;
  border-top: 1px solid #1d4ed8;
}

/* Step boundary — end */
.log-row--step-end {
  background: #16213a;
  color: #7dd3fc;
  border-bottom: 1px solid #1d4ed8;
}

/* Run start */
.log-row--run-start {
  background: #0c1a2e;
  color: #475569;
  font-style: italic;
}

/* Run end / terminal */
.log-row--run-end {
  background: #1a1a1a;
  border-top: 1px solid #334155;
  font-weight: 700;
  font-size: 13px;
}

.log-row__step-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-row__text {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: pre;
}

.log-row__ts {
  font-size: 10px;
  color: #475569;
  flex-shrink: 0;
}

.log-row__dur {
  font-size: 10px;
  color: #64748b;
  margin-left: 4px;
}

.log-row__ok {
  color: #86efac;
}

.log-row__fail {
  color: #fca5a5;
}

.log-row__terminal {
  color: #fde68a;
}
</style>
