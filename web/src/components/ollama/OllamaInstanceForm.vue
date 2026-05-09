<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { OllamaInstance } from '@/types/api'

const props = defineProps<{
  initial?: OllamaInstance | null
  existingNames?: string[]
}>()

const emit = defineEmits<{
  submit: [payload: OllamaInstance]
  cancel: []
}>()

const name = ref(props.initial?.name ?? '')
const base_url = ref(props.initial?.base_url ?? '')
const api_key = ref(props.initial?.api_key ?? '')
const errors = ref<Record<string, string>>({})

const isEdit = !!props.initial

watch(
  () => props.initial,
  (val) => {
    name.value = val?.name ?? ''
    base_url.value = val?.base_url ?? ''
    api_key.value = val?.api_key ?? ''
    errors.value = {}
  },
)

function isValidUrl(val: string): boolean {
  try {
    const u = new URL(val)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

function validate(): boolean {
  const e: Record<string, string> = {}
  if (!name.value.trim()) {
    e.name = 'Name is required.'
  } else if (!isEdit && props.existingNames?.includes(name.value.trim())) {
    e.name = 'An instance with this name already exists.'
  }
  if (!base_url.value.trim()) {
    e.base_url = 'Base URL is required.'
  } else if (!isValidUrl(base_url.value.trim())) {
    e.base_url = 'Must be a valid http(s) URL.'
  }
  errors.value = e
  return Object.keys(e).length === 0
}

function handleSubmit() {
  if (!validate()) return
  emit('submit', {
    name: name.value.trim(),
    base_url: base_url.value.trim(),
    api_key: api_key.value || undefined,
  })
}
</script>

<template>
  <form class="oif" @submit.prevent="handleSubmit">
    <div class="oif-field">
      <label class="oif-label" for="oif-name">Name</label>
      <input
        id="oif-name"
        v-model="name"
        class="oif-input"
        :class="{ 'oif-input--error': errors.name }"
        type="text"
        placeholder="e.g. local"
        :disabled="isEdit"
        autocomplete="off"
      />
      <p v-if="errors.name" class="oif-error">{{ errors.name }}</p>
    </div>

    <div class="oif-field">
      <label class="oif-label" for="oif-url">Base URL</label>
      <input
        id="oif-url"
        v-model="base_url"
        class="oif-input"
        :class="{ 'oif-input--error': errors.base_url }"
        type="text"
        placeholder="http://localhost:11434"
        autocomplete="off"
      />
      <p v-if="errors.base_url" class="oif-error">{{ errors.base_url }}</p>
    </div>

    <div class="oif-field">
      <label class="oif-label" for="oif-key">API Key <span class="oif-optional">(optional)</span></label>
      <input
        id="oif-key"
        v-model="api_key"
        class="oif-input"
        type="password"
        placeholder="Leave blank if not required"
        autocomplete="new-password"
      />
      <p v-if="isEdit && props.initial?.api_key" class="oif-hint">
        Currently set — enter a new value to replace, or leave blank to keep existing.
      </p>
    </div>

    <div class="oif-actions">
      <button type="button" class="btn-secondary" @click="emit('cancel')">Cancel</button>
      <button type="submit" class="btn-primary">{{ isEdit ? 'Save Changes' : 'Add Instance' }}</button>
    </div>
  </form>
</template>

<style scoped>
.oif {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.oif-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.oif-label {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}
.oif-optional {
  font-weight: 400;
  color: var(--color-text-muted);
}
.oif-input {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  outline: none;
}
.oif-input:focus { border-color: var(--color-accent); }
.oif-input--error { border-color: var(--color-error); }
.oif-input:disabled { opacity: 0.6; cursor: not-allowed; }
.oif-error {
  font-size: 12px;
  color: var(--color-error);
  margin: 0;
}
.oif-hint {
  font-size: 12px;
  color: var(--color-text-muted);
  margin: 0;
}
.oif-actions {
  display: flex;
  gap: var(--space-2);
  justify-content: flex-end;
  padding-top: var(--space-2);
}
.btn-primary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-primary:hover { opacity: 0.88; }
.btn-secondary {
  padding: var(--space-2) var(--space-4);
  background: transparent;
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-secondary:hover { border-color: var(--color-text-muted); color: var(--color-text); }
</style>
