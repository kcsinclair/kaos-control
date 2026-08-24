<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Provider } from '@/types/api'
import { testConnection } from '@/api/providers'

const props = defineProps<{
  initial?: Provider | null
  existingNames?: string[]
}>()

const emit = defineEmits<{
  submit: [payload: Provider]
  cancel: []
}>()

const isEdit = !!props.initial

const name = ref(props.initial?.name ?? '')
const base_url = ref(props.initial?.base_url ?? '')
const driver = ref(props.initial?.driver ?? 'openai-compatible')
const api_key = ref(props.initial?.api_key && props.initial.api_key !== '***' ? props.initial.api_key : '')
const showApiKey = ref(false)

interface HeaderRow {
  key: string
  value: string
}

function initHeaders(headersMap?: Record<string, string>): HeaderRow[] {
  if (!headersMap) return []
  return Object.entries(headersMap).map(([key, value]) => ({ key, value }))
}

const headers = ref<HeaderRow[]>(initHeaders(props.initial?.extra_headers))

function addHeader() {
  headers.value.push({ key: '', value: '' })
}

function removeHeader(index: number) {
  headers.value.splice(index, 1)
}

const errors = ref<Record<string, string>>({})

// Testing connection state
const testing = ref(false)
const testResult = ref<{ ok: boolean; latency_ms?: number; message?: string; error?: string } | null>(null)

watch(
  () => props.initial,
  (val) => {
    name.value = val?.name ?? ''
    base_url.value = val?.base_url ?? ''
    driver.value = val?.driver ?? 'openai-compatible'
    api_key.value = val?.api_key && val.api_key !== '***' ? val.api_key : ''
    headers.value = initHeaders(val?.extra_headers)
    errors.value = {}
    testResult.value = null
  },
)

interface Preset {
  label: string
  name: string
  base_url: string
  driver: string
  extra_headers?: Record<string, string>
}

const presets: Preset[] = [
  { label: 'Ollama Local (http://localhost:11434)', name: 'ollama', base_url: 'http://localhost:11434', driver: 'openai-compatible' },
  { label: 'llama.cpp (http://localhost:7442)', name: 'llama-cpp', base_url: 'http://localhost:7442', driver: 'openai-compatible' },
  {
    label: 'OpenRouter (https://openrouter.ai/api/v1)',
    name: 'openrouter',
    base_url: 'https://openrouter.ai/api/v1',
    driver: 'openai-compatible',
    extra_headers: { 'HTTP-Referer': 'https://kaos-control.local', 'X-Title': 'kaos-control' },
  },
  { label: 'OpenAI (https://api.openai.com/v1)', name: 'openai', base_url: 'https://api.openai.com/v1', driver: 'openai-compatible' },
]

function applyPreset(p: Preset) {
  if (!isEdit && !name.value.trim()) {
    name.value = p.name
  }
  base_url.value = p.base_url
  driver.value = p.driver
  if (p.extra_headers) {
    headers.value = initHeaders(p.extra_headers)
  }
}

function isValidSlug(val: string): boolean {
  return /^[a-z0-9-]+$/.test(val)
}

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
  const trimmedName = name.value.trim()
  if (!trimmedName) {
    e.name = 'Name is required.'
  } else if (!isValidSlug(trimmedName)) {
    e.name = 'Name must contain only lowercase letters, numbers, and hyphens.'
  } else if (!isEdit && props.existingNames?.includes(trimmedName)) {
    e.name = 'A provider with this name already exists.'
  }
  if (!base_url.value.trim()) {
    e.base_url = 'Base URL is required.'
  } else if (!isValidUrl(base_url.value.trim())) {
    e.base_url = 'Must be a valid http(s) URL.'
  }
  if (!driver.value.trim()) {
    e.driver = 'Driver is required.'
  }
  errors.value = e
  return Object.keys(e).length === 0
}

function collectHeaders(): Record<string, string> | undefined {
  const result: Record<string, string> = {}
  for (const h of headers.value) {
    const k = h.key.trim()
    if (k) {
      result[k] = h.value.trim()
    }
  }
  return Object.keys(result).length ? result : undefined
}

async function handleTest() {
  if (!base_url.value.trim()) {
    errors.value = { base_url: 'Enter a base URL to test connection.' }
    return
  }
  testing.value = true
  testResult.value = null
  try {
    const res = await testConnection({
      name: name.value.trim() || undefined,
      base_url: base_url.value.trim(),
      driver: driver.value.trim() || 'openai-compatible',
      api_key: api_key.value ? api_key.value.trim() : undefined,
      extra_headers: collectHeaders(),
    })
    testResult.value = res
  } catch (err: unknown) {
    testResult.value = {
      ok: false,
      error: err instanceof Error ? err.message : 'Connection test failed',
    }
  } finally {
    testing.value = false
  }
}

function handleSubmit() {
  if (!validate()) return
  const payload: Provider = {
    name: name.value.trim(),
    base_url: base_url.value.trim(),
    driver: driver.value.trim() || 'openai-compatible',
    api_key: api_key.value ? api_key.value.trim() : undefined,
    extra_headers: collectHeaders(),
  }
  emit('submit', payload)
}
</script>

<template>
  <form class="pvf" @submit.prevent="handleSubmit">
    <!-- Presets (Add mode only) -->
    <div v-if="!isEdit" class="pvf-field">
      <div class="pvf-label">Quick Presets</div>
      <div class="pvf-presets">
        <button
          v-for="p in presets"
          :key="p.label"
          type="button"
          class="pvf-preset-chip"
          @click="applyPreset(p)"
        >
          {{ p.label }}
        </button>
      </div>
    </div>

    <!-- Name -->
    <div class="pvf-field">
      <label class="pvf-label" for="pvf-name">Name</label>
      <input
        id="pvf-name"
        v-model="name"
        class="pvf-input"
        :class="{ 'pvf-input--error': errors.name }"
        type="text"
        placeholder="e.g. llama-cpp or openrouter"
        :disabled="isEdit"
        autocomplete="off"
      />
      <p v-if="errors.name" class="pvf-error">{{ errors.name }}</p>
    </div>

    <!-- Base URL -->
    <div class="pvf-field">
      <label class="pvf-label" for="pvf-url">Base URL</label>
      <input
        id="pvf-url"
        v-model="base_url"
        class="pvf-input"
        :class="{ 'pvf-input--error': errors.base_url }"
        type="text"
        placeholder="e.g. http://leia.packsin.com:7442 or https://openrouter.ai/api"
        autocomplete="off"
      />
      <p v-if="errors.base_url" class="pvf-error">{{ errors.base_url }}</p>
    </div>

    <!-- Driver -->
    <div class="pvf-field">
      <label class="pvf-label" for="pvf-driver">Driver</label>
      <select id="pvf-driver" v-model="driver" class="pvf-select">
        <option value="openai-compatible">openai-compatible (OpenAI / llama.cpp / Ollama / OpenRouter)</option>
      </select>
    </div>

    <!-- API Key -->
    <div class="pvf-field">
      <label class="pvf-label" for="pvf-key">
        API Key <span class="pvf-optional">(optional)</span>
      </label>
      <div class="pvf-input-group">
        <input
          id="pvf-key"
          v-model="api_key"
          class="pvf-input"
          :type="showApiKey ? 'text' : 'password'"
          :placeholder="isEdit && (props.initial?.has_api_key || props.initial?.api_key) ? '••••••••' : 'Leave blank if not required'"
          autocomplete="new-password"
        />
        <button
          type="button"
          class="btn-toggle-key"
          @click="showApiKey = !showApiKey"
        >
          {{ showApiKey ? 'Hide' : 'Show' }}
        </button>
      </div>
      <p v-if="isEdit && props.initial?.has_api_key" class="pvf-hint">
        An API key is currently saved. Enter a new key to update, or leave blank to keep unchanged.
      </p>
    </div>

    <!-- Extra Headers -->
    <div class="pvf-field">
      <div class="pvf-label-row">
        <span class="pvf-label">Extra Headers <span class="pvf-optional">(e.g. OpenRouter attribution)</span></span>
        <button type="button" class="btn-add-header" @click="addHeader">+ Add Header</button>
      </div>
      <div v-if="headers.length" class="pvf-headers-list">
        <div v-for="(h, idx) in headers" :key="idx" class="pvf-header-row">
          <input
            v-model="h.key"
            class="pvf-input pvf-header-key"
            type="text"
            placeholder="Header Name (e.g. HTTP-Referer)"
          />
          <input
            v-model="h.value"
            class="pvf-input pvf-header-val"
            type="text"
            placeholder="Header Value"
          />
          <button
            type="button"
            class="btn-remove-header"
            title="Remove header"
            @click="removeHeader(idx)"
          >
            ✕
          </button>
        </div>
      </div>
    </div>

    <!-- Test Connection -->
    <div class="pvf-test-section">
      <div class="pvf-test-row">
        <button
          type="button"
          class="btn-test"
          :disabled="testing || !base_url"
          @click="handleTest"
        >
          {{ testing ? 'Testing Connection…' : 'Test Connection' }}
        </button>
        <span v-if="testResult" class="test-status" :class="testResult.ok ? 'test-status--ok' : 'test-status--err'">
          {{ testResult.ok ? `✓ Connected (${testResult.latency_ms ?? 0} ms)` : `✗ ${testResult.error || 'Failed'}` }}
        </span>
      </div>
    </div>

    <div class="pvf-actions">
      <button type="button" class="btn-secondary" @click="emit('cancel')">Cancel</button>
      <button type="submit" class="btn-primary">{{ isEdit ? 'Save Changes' : 'Add Provider' }}</button>
    </div>
  </form>
</template>

<style scoped>
.pvf {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.pvf-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.pvf-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.pvf-label {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}
.pvf-optional {
  font-weight: 400;
  color: var(--color-text-muted);
  font-size: 12px;
}
.pvf-presets {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}
.pvf-preset-chip {
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  border-radius: 99px;
  font-size: 12px;
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
  transition: border-color 0.1s, background 0.1s;
}
.pvf-preset-chip:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}
.pvf-input {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  outline: none;
}
.pvf-input:focus { border-color: var(--color-accent); }
.pvf-input--error { border-color: var(--color-error); }
.pvf-input:disabled { opacity: 0.6; cursor: not-allowed; }
.pvf-select {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  outline: none;
}
.pvf-input-group {
  display: flex;
  gap: var(--space-2);
}
.pvf-input-group .pvf-input {
  flex: 1;
}
.btn-toggle-key {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-text-muted);
  font-size: 12px;
  cursor: pointer;
}
.btn-toggle-key:hover {
  color: var(--color-text);
  border-color: var(--color-text-muted);
}
.pvf-error {
  font-size: 12px;
  color: var(--color-error);
  margin: 0;
}
.pvf-hint {
  font-size: 12px;
  color: var(--color-text-muted);
  margin: 0;
}
.btn-add-header {
  background: none;
  border: none;
  color: var(--color-accent);
  font-size: 12px;
  cursor: pointer;
  padding: 0;
}
.btn-add-header:hover { text-decoration: underline; }
.pvf-headers-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-top: var(--space-1);
}
.pvf-header-row {
  display: flex;
  gap: var(--space-2);
  align-items: center;
}
.pvf-header-key {
  flex: 2;
  font-family: monospace;
  font-size: 12px;
}
.pvf-header-val {
  flex: 3;
  font-family: monospace;
  font-size: 12px;
}
.btn-remove-header {
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
  padding: 4px;
  font-size: 14px;
}
.btn-remove-header:hover { color: var(--color-error); }
.pvf-test-section {
  padding: var(--space-2) 0;
  border-top: 1px solid var(--color-border);
  border-bottom: 1px solid var(--color-border);
}
.pvf-test-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.btn-test {
  padding: var(--space-1) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
}
.btn-test:hover:not(:disabled) {
  border-color: var(--color-accent);
  color: var(--color-accent);
}
.btn-test:disabled { opacity: 0.5; cursor: not-allowed; }
.test-status {
  font-size: 12px;
  font-weight: 500;
}
.test-status--ok { color: #16a34a; }
.test-status--err { color: #dc2626; }
.pvf-actions {
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
