<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, watch, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { patchRelease } from '@/api/artifacts'
import { listReleases } from '@/api/releases'
import type { Release } from '@/types/release'

const props = defineProps<{
  project: string
  path: string
  release: string | null
  readonly?: boolean
}>()

const emit = defineEmits<{
  changed: [newRelease: string | null]
  error: [message: string]
}>()

// ── state ──────────────────────────────────────────────────────────────────────
const isOpen = ref(false)
const optimisticRelease = ref<string | null>(props.release)
const focusedIndex = ref(-1)

const releases = ref<Release[]>([])
const releasesCached = ref(false)

const wrapRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)

// ── options list: "None" + all project releases ────────────────────────────────
// Index 0 is always "None"; indices 1..N map to releases[i-1]
const optionCount = computed(() => 1 + releases.value.length)

// ── watch prop for external WebSocket updates ─────────────────────────────────
watch(() => props.release, (newVal) => {
  if (!isOpen.value) {
    optimisticRelease.value = newVal
  }
})

// ── aria helpers ──────────────────────────────────────────────────────────────
const activeDescendant = computed(() =>
  focusedIndex.value >= 0 ? `release-opt-${focusedIndex.value}` : undefined,
)

// ── open / close ──────────────────────────────────────────────────────────────
async function openMenu() {
  if (isOpen.value || props.readonly) return

  // Fetch releases unless already cached for this component's lifetime
  if (!releasesCached.value) {
    try {
      releases.value = await listReleases(props.project)
      releasesCached.value = true
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to load releases'
      emit('error', msg)
      return
    }
  }

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
// index 0 = "None", index i (1-based) = releases[i-1]
function releaseAtIndex(i: number): string | null {
  if (i === 0) return null
  return releases.value[i - 1]?.name ?? null
}

async function select(index: number) {
  const value = releaseAtIndex(index)
  if (value === optimisticRelease.value) {
    closeMenu()
    triggerRef.value?.focus()
    return
  }

  const previous = optimisticRelease.value
  optimisticRelease.value = value   // optimistic update
  closeMenu()
  triggerRef.value?.focus()

  try {
    await patchRelease(props.project, props.path, value)
    emit('changed', value)
  } catch (e: unknown) {
    optimisticRelease.value = previous   // revert on failure
    const msg = e instanceof Error ? e.message : 'Release update failed'
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
    const next = Math.min(focusedIndex.value + 1, optionCount.value - 1)
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
    if (focusedIndex.value >= 0 && focusedIndex.value < optionCount.value) {
      select(focusedIndex.value)
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
  <span ref="wrapRef" class="release-dropdown-wrap">

    <!-- Read-only badge -->
    <span
      v-if="readonly"
      class="release-badge"
      :class="{ 'release-badge--none': !optimisticRelease }"
    >{{ optimisticRelease ?? 'None' }}</span>

    <!-- Interactive trigger -->
    <button
      v-else
      ref="triggerRef"
      class="release-badge release-badge--interactive"
      :class="{ 'release-badge--none': !optimisticRelease }"
      type="button"
      role="button"
      aria-haspopup="listbox"
      :aria-expanded="isOpen ? 'true' : 'false'"
      tabindex="0"
      @click="isOpen ? closeMenu() : openMenu()"
      @keydown="onTriggerKeydown"
    >{{ optimisticRelease ?? 'None' }}</button>

    <!-- Dropdown panel -->
    <div
      v-if="isOpen"
      ref="menuRef"
      class="release-menu"
      role="listbox"
      :aria-label="`Change release`"
      :aria-activedescendant="activeDescendant"
      @keydown="onMenuKeydown"
    >
      <!-- None option -->
      <div
        id="release-opt-0"
        class="release-option release-option--none"
        role="option"
        :aria-selected="optimisticRelease === null"
        tabindex="-1"
        @click="select(0)"
        @mouseenter="focusedIndex = 0"
      >
        <span class="release-none-label">None</span>
      </div>

      <!-- Release options -->
      <div
        v-for="(rel, i) in releases"
        :id="`release-opt-${i + 1}`"
        :key="rel.id"
        class="release-option"
        role="option"
        :aria-selected="rel.name === optimisticRelease"
        tabindex="-1"
        @click="select(i + 1)"
        @mouseenter="focusedIndex = i + 1"
      >
        <span class="release-name">{{ rel.name }}</span>
        <span class="release-status">{{ rel.status }}</span>
      </div>
    </div>

  </span>
</template>

<style scoped>
/* ── wrapper ─────────────────────────────────────────────────────────────────*/
.release-dropdown-wrap {
  position: relative;
  display: inline-block;
}

/* ── badge ───────────────────────────────────────────────────────────────────*/
.release-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  border: 1px solid var(--color-border);
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  user-select: none;
  line-height: 1.6;
  background: var(--color-surface-raised, rgba(0, 0, 0, 0.04));
  color: var(--color-text);
}

.release-badge--none {
  color: var(--color-text-muted);
  font-style: italic;
}

/* Reset button defaults so it matches the <span> badge exactly */
.release-badge--interactive {
  cursor: pointer;
  font-family: inherit;
  text-align: left;
}
.release-badge--interactive:hover {
  opacity: 0.85;
}
.release-badge--interactive:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* ── dropdown panel ──────────────────────────────────────────────────────────*/
.release-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 100;
  min-width: 160px;
  max-width: min(240px, calc(100vw - 16px));
  background: var(--color-surface, #fff);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md, 6px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  padding: 4px 0;
  outline: none;
}

/* ── option rows ─────────────────────────────────────────────────────────────*/
.release-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 5px 10px;
  cursor: pointer;
  font-size: 12px;
  color: var(--color-text);
}
.release-option:hover,
.release-option:focus {
  background: var(--color-surface-raised, rgba(0, 0, 0, 0.05));
  outline: none;
}

.release-option--none {
  border-bottom: 1px solid var(--color-border);
  margin-bottom: 2px;
}

.release-none-label {
  color: var(--color-text-muted);
  font-style: italic;
}

.release-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.release-status {
  flex-shrink: 0;
  font-size: 10px;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
</style>
