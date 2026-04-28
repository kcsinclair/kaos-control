<script setup lang="ts">
import { X } from 'lucide-vue-next'
import type { ArtifactAssignee } from '@/types/api'

const props = defineProps<{
  modelValue: ArtifactAssignee[]
  roles: string[]
  whoOptions: string[]
}>()

const emit = defineEmits<{ 'update:modelValue': [v: ArtifactAssignee[]] }>()

const listId = `who-options-${Math.random().toString(36).slice(2)}`

function updateRole(index: number, role: string) {
  const updated = props.modelValue.map((a, i) => (i === index ? { ...a, role } : a))
  emit('update:modelValue', updated)
}

function updateWho(index: number, who: string) {
  const updated = props.modelValue.map((a, i) => (i === index ? { ...a, who } : a))
  emit('update:modelValue', updated)
}

function remove(index: number) {
  const updated = props.modelValue.filter((_, i) => i !== index)
  emit('update:modelValue', updated)
}

function add() {
  emit('update:modelValue', [...props.modelValue, { role: '', who: '' }])
}
</script>

<template>
  <div class="ae-root">
    <datalist :id="listId">
      <option v-for="opt in whoOptions" :key="opt" :value="opt" />
    </datalist>

    <div
      v-for="(assignee, i) in modelValue"
      :key="i"
      class="ae-row"
    >
      <select
        class="ae-select"
        :value="assignee.role"
        aria-label="Role"
        @change="updateRole(i, ($event.target as HTMLSelectElement).value)"
      >
        <option value="">— role —</option>
        <option v-for="r in roles" :key="r" :value="r">{{ r }}</option>
      </select>

      <input
        class="ae-input"
        type="text"
        :value="assignee.who"
        :list="listId"
        aria-label="Assignee"
        placeholder="who"
        @input="updateWho(i, ($event.target as HTMLInputElement).value)"
      />

      <button type="button" class="ae-remove" aria-label="Remove assignee" @click="remove(i)">
        <X :size="14" />
      </button>
    </div>

    <button type="button" class="ae-add" @keydown.enter.prevent="add" @click="add">
      + Add assignee
    </button>
  </div>
</template>

<style scoped>
.ae-root {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.ae-row {
  display: flex;
  gap: var(--space-1);
  align-items: center;
}
.ae-select,
.ae-input {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  box-sizing: border-box;
  min-width: 0;
}
.ae-select {
  flex: 1 1 0;
  appearance: none;
  -webkit-appearance: none;
  cursor: pointer;
}
.ae-input {
  flex: 1 1 0;
}
.ae-select:focus,
.ae-input:focus {
  outline: none;
  border-color: var(--color-accent);
}
.ae-remove {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
}
.ae-remove:hover {
  background: var(--color-border);
  color: var(--color-text);
}
.ae-add {
  align-self: flex-start;
  padding: 2px var(--space-2);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  font-family: inherit;
  cursor: pointer;
}
.ae-add:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}
</style>
