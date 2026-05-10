<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, watch, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { patchPriority } from '@/api/artifacts'
import { useGraphTheme } from '@/components/map/graphConstants'

const props = defineProps<{
  project: string
  path: string
  priority: string
  readonly?: boolean
}>()

const emit = defineEmits<{
  changed: [newPriority: string]
  error: [message: string]
}>()

const { palette } = useGraphTheme()

const PRIORITY_OPTIONS = ['high', 'medium', 'normal', 'low']

// ── state ─────────────────────────────────────────────────────────────────────
const isOpen = ref(false)
const optimisticPriority = ref(props.priority || 'normal')
const focusedIndex = ref(-1)

const wrapRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)

// ── colour helpers ────────────────────────────────────────────────────────────
function badgeStyle(p: string) {
  const color = palette.value.priorityColors[p] ?? '#6b7280'
  return {
    background: color + '33',
    color,
    borderColor: color + '66',
  }
}

function dotColor(p: string): string {
  return palette.value.priorityColors[p] ?? '#6b7280'
}

// ── aria helpers ──────────────────────────────────────────────────────────────
const activeDescendant = computed(() =>
  focusedIndex.value >= 0 ? `priority-opt-${focusedIndex.value}` : undefined,
)

// ── watch prop for external WebSocket updates ─────────────────────────────────
watch(() => props.priority, (newVal) => {
  if (!isOpen.value) {
    optimisticPriority.value = newVal || 'normal'
  }
})

// ── open / close ──────────────────────────────────────────────────────────────
async function openMenu() {
  if (isOpen.value || props.readonly) return
  isOpen.value = true
  focusedIndex.value = 0
  await nextTick()
  focusOptionAt(0)
}

function closeMenu() {
  isOpen.value = false
  focusedIndex.value = -1
}

function focusOptionAt(index: number) {
  if (!menuRef.value) return
  const options = menuRef.value.querySelectorAll<HTMLElement>('[role="option"]')
  if (options[index]) {
    options[index].focus()
    focusedIndex.value = index
  }
}

// ── select ────────────────────────────────────────────────────────────────────
async function select(value: string) {
  if (value === optimisticPriority.value) {
    closeMenu()
    triggerRef.value?.focus()
    return
  }

  const previous = optimisticPriority.value
  optimisticPriority.value = value   // optimistic update
  closeMenu()
  triggerRef.value?.focus()

  try {
    await patchPriority(props.project, props.path, value)
    emit('changed', value)
  } catch (e: unknown) {
    optimisticPriority.value = previous   // revert on failure
    const msg = e instanceof Error ? e.message : 'Priority update failed'
    emit('error', msg)
  }
}

// ── keyboard: trigger ─────────────────────────────────────────────────────────
function onTriggerKeydown(e: KeyboardEvent) {
  if (props.readonly) return
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    if (!isOpen.value) openMenu()
    else closeMenu()
  }
  if (e.key === 'Escape' && isOpen.value) {
    e.preventDefault()
    closeMenu()
  }
}

// ── keyboard: menu ────────────────────────────────────────────────────────────
function onMenuKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    closeMenu()
    triggerRef.value?.focus()
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    const next = Math.min(focusedIndex.value + 1, PRIORITY_OPTIONS.length - 1)
    focusedIndex.value = next
    focusOptionAt(next)
    return
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    const prev = Math.max(focusedIndex.value - 1, 0)
    focusedIndex.value = prev
    focusOptionAt(prev)
    return
  }
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    if (focusedIndex.value >= 0 && focusedIndex.value < PRIORITY_OPTIONS.length) {
      select(PRIORITY_OPTIONS[focusedIndex.value])
    }
  }
}

// ── outside click ─────────────────────────────────────────────────────────────
function onDocumentClick(e: MouseEvent) {
  if (!isOpen.value) return
  if (wrapRef.value?.contains(e.target as Node)) return
  closeMenu()
}

onMounted(() => document.addEventListener('click', onDocumentClick, true))
onBeforeUnmount(() => document.removeEventListener('click', onDocumentClick, true))
</script>

<template>
  <span ref="wrapRef" class="priority-dropdown-wrap">

    <!-- Read-only badge (no project/path or locked by another user) -->
    <span
      v-if="readonly"
      class="priority-badge"
      :style="badgeStyle(optimisticPriority)"
    >{{ optimisticPriority }}</span>

    <!-- Interactive trigger -->
    <button
      v-else
      ref="triggerRef"
      class="priority-badge priority-badge--interactive"
      :style="badgeStyle(optimisticPriority)"
      type="button"
      role="button"
      aria-haspopup="listbox"
      :aria-expanded="isOpen ? 'true' : 'false'"
      tabindex="0"
      @click="isOpen ? closeMenu() : openMenu()"
      @keydown="onTriggerKeydown"
    >{{ optimisticPriority }}</button>

    <!-- Dropdown panel -->
    <div
      v-if="isOpen"
      ref="menuRef"
      class="priority-menu"
      role="listbox"
      :aria-label="`Change priority from ${optimisticPriority}`"
      :aria-activedescendant="activeDescendant"
      @keydown="onMenuKeydown"
    >
      <div
        v-for="(opt, i) in PRIORITY_OPTIONS"
        :id="`priority-opt-${i}`"
        :key="opt"
        class="priority-option"
        role="option"
        :aria-selected="opt === optimisticPriority"
        tabindex="-1"
        @click="select(opt)"
        @mouseenter="focusedIndex = i"
      >
        <span class="priority-dot" :style="{ background: dotColor(opt) }" aria-hidden="true" />
        <span>{{ opt }}</span>
      </div>
    </div>

  </span>
</template>

<style scoped>
/* ── wrapper ─────────────────────────────────────────────────────────────────*/
.priority-dropdown-wrap {
  position: relative;
  display: inline-block;
}

/* ── badge ───────────────────────────────────────────────────────────────────*/
.priority-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  border: 1px solid transparent;
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  user-select: none;
  line-height: 1.6;
}

/* Reset button defaults so it matches the <span> badge exactly */
.priority-badge--interactive {
  cursor: pointer;
  background: none;  /* overridden by :style */
  font-family: inherit;
  text-align: left;
}
.priority-badge--interactive:hover {
  opacity: 0.85;
}
.priority-badge--interactive:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* ── dropdown panel ──────────────────────────────────────────────────────────*/
.priority-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 100;
  min-width: 130px;
  max-width: min(200px, calc(100vw - 16px));
  background: var(--color-surface, #fff);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md, 6px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  padding: 4px 0;
  outline: none;
}

/* ── option rows ─────────────────────────────────────────────────────────────*/
.priority-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  cursor: pointer;
  font-size: 12px;
  color: var(--color-text);
}
.priority-option:hover,
.priority-option:focus {
  background: var(--color-surface-raised, rgba(0, 0, 0, 0.05));
  outline: none;
}

/* ── colour dot ──────────────────────────────────────────────────────────────*/
.priority-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
</style>
