<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { Search, X } from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  debounceMs?: number
}>(), {
  placeholder: 'Filter by text…',
  debounceMs: 200,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const inputRef = ref<HTMLInputElement | null>(null)
const expanded = ref(false)

let debounceTimer: ReturnType<typeof setTimeout> | null = null

function onInput(e: Event) {
  const value = (e.target as HTMLInputElement).value
  if (debounceTimer !== null) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    emit('update:modelValue', value)
  }, props.debounceMs)
}

function onClear() {
  if (debounceTimer !== null) {
    clearTimeout(debounceTimer)
    debounceTimer = null
  }
  emit('update:modelValue', '')
  // On mobile, collapse after clear
  expanded.value = false
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (debounceTimer !== null) {
      clearTimeout(debounceTimer)
      debounceTimer = null
    }
    emit('update:modelValue', '')
    expanded.value = false
    inputRef.value?.blur()
  }
}

function onBlur() {
  if (!props.modelValue) {
    expanded.value = false
  }
}

function focus() {
  expanded.value = true
  nextTick(() => inputRef.value?.focus())
}

defineExpose({ focus })
</script>

<template>
  <div class="text-filter" :class="{ 'text-filter--open': expanded || !!modelValue }">
    <!-- Mobile-only toggle button (hidden on desktop via CSS) -->
    <button
      class="text-filter__toggle"
      aria-label="Open text filter"
      @click="focus"
    >
      <Search :size="15" />
    </button>

    <!-- Input row (always visible on desktop, toggled on mobile) -->
    <div class="text-filter__body">
      <Search class="text-filter__icon" :size="14" aria-hidden="true" />
      <input
        ref="inputRef"
        class="text-filter__input"
        type="text"
        :value="modelValue"
        :placeholder="placeholder"
        aria-label="Filter artifacts by text"
        @input="onInput"
        @keydown="onKeydown"
        @blur="onBlur"
      />
      <button
        v-if="modelValue"
        class="text-filter__clear"
        aria-label="Clear filter"
        tabindex="0"
        @click="onClear"
      >
        <X :size="12" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.text-filter {
  display: inline-flex;
  align-items: center;
  position: relative;
}

/* Mobile toggle: hidden by default on desktop */
.text-filter__toggle {
  display: none;
}

.text-filter__body {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  padding: 0 var(--space-2);
}

.text-filter__icon {
  color: var(--color-text-muted);
  flex-shrink: 0;
}

.text-filter__input {
  background: none;
  border: none;
  outline: none;
  font-size: var(--text-sm);
  color: var(--color-text);
  min-width: 140px;
  padding: var(--space-1) 0;
}

.text-filter__input::placeholder {
  color: var(--color-text-muted);
}

.text-filter__clear {
  background: none;
  border: none;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  padding: 0;
  color: var(--color-text-muted);
  flex-shrink: 0;
}

.text-filter__clear:hover {
  color: var(--color-text);
}

.text-filter__clear:focus-visible {
  outline: 2px solid var(--color-accent);
  border-radius: 2px;
  outline-offset: 1px;
}

/* ── Mobile collapse (≤ 480px) ──────────────────────────── */
@media (max-width: 480px) {
  /* Show toggle icon, hide input by default */
  .text-filter__toggle {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    padding: var(--space-1);
    cursor: pointer;
    color: var(--color-text-muted);
  }

  .text-filter__toggle:focus-visible {
    outline: 2px solid var(--color-accent);
    outline-offset: 2px;
  }

  .text-filter__body {
    display: none;
  }

  /* When expanded (or has value): show input, hide toggle */
  .text-filter--open .text-filter__toggle {
    display: none;
  }

  .text-filter--open .text-filter__body {
    display: inline-flex;
  }
}
</style>
