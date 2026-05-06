<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { getAllowedTargets, transitionArtifact } from '@/api/artifacts'

const props = defineProps<{
  project: string
  path: string
  status: string
}>()

const emit = defineEmits<{
  transitioned: [newStatus: string]
  error: [message: string]
}>()

// ── state ────────────────────────────────────────────────────────────────────
const isOpen = ref(false)
const loading = ref(false)
const targets = ref<string[]>([])
const optimisticStatus = ref(props.status)
const focusedIndex = ref(-1)
/** true after at least one successful fetch; used to decide if badge is interactive */
const hasFetched = ref(false)

const wrapRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)

// ── watch prop for external WS status changes ─────────────────────────────────
// ArtifactEditorView re-fetches the artifact on artifact.indexed WS events and
// updates artifact.value, which flows down as the :status prop here.
// If the dropdown is open when the prop changes, we close it so the user sees
// the up-to-date status rather than acting on stale data.
watch(() => props.status, (newVal) => {
  if (isOpen.value) {
    // Another user/agent transitioned while dropdown was open — close and reset
    closeMenu()
  }
  optimisticStatus.value = newVal
  // Reset hasFetched so the badge is interactive again for the new status
  hasFetched.value = false
})

// ── open / close ─────────────────────────────────────────────────────────────
async function openMenu() {
  if (isOpen.value) return
  loading.value = true
  targets.value = []
  isOpen.value = true
  focusedIndex.value = -1

  try {
    const res = await getAllowedTargets(props.project, props.path)
    targets.value = res.targets ?? []
  } catch {
    targets.value = []
  } finally {
    loading.value = false
    hasFetched.value = true
  }

  await nextTick()
  focusFirstOption()
}

function closeMenu() {
  isOpen.value = false
  loading.value = false
  focusedIndex.value = -1
}

function focusFirstOption() {
  if (!menuRef.value) return
  const options = menuRef.value.querySelectorAll<HTMLElement>('[role="option"]')
  if (options.length) {
    options[0].focus()
    focusedIndex.value = 0
  }
}

function focusOptionAt(index: number) {
  if (!menuRef.value) return
  const options = menuRef.value.querySelectorAll<HTMLElement>('[role="option"]')
  options[index]?.focus()
}

// ── select a target ───────────────────────────────────────────────────────────
async function select(target: string) {
  const previous = optimisticStatus.value
  optimisticStatus.value = target   // optimistic update
  closeMenu()
  triggerRef.value?.focus()

  try {
    await transitionArtifact(props.project, props.path, target)
    emit('transitioned', target)
  } catch (e: unknown) {
    optimisticStatus.value = previous   // revert
    const msg = e instanceof Error ? e.message : 'Transition failed'
    emit('error', msg)
  }
}

// ── keyboard on trigger ───────────────────────────────────────────────────────
function onTriggerKeydown(e: KeyboardEvent) {
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

// ── keyboard on menu ──────────────────────────────────────────────────────────
function onMenuKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    closeMenu()
    triggerRef.value?.focus()
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    const next = Math.min(focusedIndex.value + 1, targets.value.length - 1)
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
    if (focusedIndex.value >= 0 && focusedIndex.value < targets.value.length) {
      select(targets.value[focusedIndex.value])
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
  <span ref="wrapRef" class="status-dropdown-wrap">
    <!-- Non-interactive: fetched and no transitions available -->
    <span
      v-if="hasFetched && !isOpen && targets.length === 0"
      class="status-badge"
      :data-status="optimisticStatus"
      title="No transitions available"
    >{{ optimisticStatus }}</span>

    <!-- Interactive trigger -->
    <span
      v-else
      ref="triggerRef"
      class="status-badge status-badge--interactive"
      :data-status="optimisticStatus"
      role="button"
      aria-haspopup="listbox"
      :aria-expanded="isOpen ? 'true' : 'false'"
      tabindex="0"
      @click="isOpen ? closeMenu() : openMenu()"
      @keydown="onTriggerKeydown"
    >{{ optimisticStatus }}</span>

    <!-- Dropdown menu -->
    <div
      v-if="isOpen"
      ref="menuRef"
      class="status-menu"
      role="listbox"
      :aria-label="`Change status from ${optimisticStatus}`"
      @keydown="onMenuKeydown"
    >
      <!-- Loading spinner -->
      <div v-if="loading" class="status-menu-loading">
        <span class="spinner" aria-hidden="true" />
        Loading…
      </div>
      <!-- Options -->
      <template v-else-if="targets.length > 0">
        <div
          v-for="(t, i) in targets"
          :key="t"
          class="status-option"
          role="option"
          :aria-selected="t === optimisticStatus"
          tabindex="-1"
          @click="select(t)"
          @mouseenter="focusedIndex = i"
        >
          <span class="status-badge status-badge--sm" :data-status="t">{{ t }}</span>
        </div>
      </template>
      <!-- No options -->
      <div v-else class="status-menu-empty">No transitions available</div>
    </div>
  </span>
</template>

<style scoped>
/* ── wrapper ────────────────────────────────────────────────────────────────── */
.status-dropdown-wrap {
  position: relative;
  display: inline-block;
}

/* ── badge ─────────────────────────────────────────────────────────────────── */
.status-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  background: var(--color-border);
  color: var(--color-text);
  white-space: nowrap;
  user-select: none;
}
.status-badge--sm {
  padding: 1px 7px;
}
.status-badge--interactive {
  cursor: pointer;
}
.status-badge--interactive:hover {
  opacity: 0.85;
}
.status-badge--interactive:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* ── status colours — light mode ─────────────────────────────────────────────*/
.status-badge[data-status="draft"]          { background: #f3f4f6; color: #374151; }
.status-badge[data-status="clarifying"]     { background: #ede9fe; color: #5b21b6; }
.status-badge[data-status="planning"]       { background: #fef3c7; color: #92400e; }
.status-badge[data-status="in-development"] { background: #dbeafe; color: #1e40af; }
.status-badge[data-status="in-qa"]          { background: #ede9fe; color: #6d28d9; }
.status-badge[data-status="approved"]       { background: #d1fae5; color: #065f46; }
.status-badge[data-status="done"]           { background: #bbf7d0; color: #14532d; }
.status-badge[data-status="blocked"]        { background: #fee2e2; color: #991b1b; }
.status-badge[data-status="rejected"]       { background: #fef2f2; color: #b91c1c; }
.status-badge[data-status="abandoned"]      { background: #f3f4f6; color: #6b7280; }
.status-badge[data-status="in-progress"]    { background: #fef3c7; color: #92400e; }

/* ── status colours — dark mode ──────────────────────────────────────────────*/
@media (prefers-color-scheme: dark) {
  .status-badge[data-status="draft"]          { background: #374151; color: #d1d5db; }
  .status-badge[data-status="clarifying"]     { background: #3b2f6e; color: #c4b5fd; }
  .status-badge[data-status="planning"]       { background: #422006; color: #fcd34d; }
  .status-badge[data-status="in-development"] { background: #1e3a5f; color: #93c5fd; }
  .status-badge[data-status="in-qa"]          { background: #2e1065; color: #c4b5fd; }
  .status-badge[data-status="approved"]       { background: #064e3b; color: #6ee7b7; }
  .status-badge[data-status="done"]           { background: #052e16; color: #4ade80; }
  .status-badge[data-status="blocked"]        { background: #7f1d1d; color: #fca5a5; }
  .status-badge[data-status="rejected"]       { background: #7f1d1d; color: #fca5a5; }
  .status-badge[data-status="abandoned"]      { background: #1f2937; color: #9ca3af; }
  .status-badge[data-status="in-progress"]    { background: #422006; color: #fcd34d; }
}

/* ── dropdown menu ───────────────────────────────────────────────────────────*/
.status-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 100;
  min-width: 160px;
  max-width: min(220px, calc(100vw - 16px));
  background: var(--color-surface, #fff);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md, 6px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.12);
  padding: 4px 0;
  outline: none;
}

.status-menu-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--color-text-muted);
}

.status-menu-empty {
  padding: 8px 12px;
  font-size: 12px;
  color: var(--color-text-muted);
}

.status-option {
  display: flex;
  align-items: center;
  padding: 5px 10px;
  cursor: pointer;
}
.status-option:hover,
.status-option:focus {
  background: var(--color-surface-raised, rgba(0,0,0,0.05));
  outline: none;
}

/* ── loading spinner ─────────────────────────────────────────────────────────*/
.spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-accent, #3b82f6);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
  flex-shrink: 0;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
