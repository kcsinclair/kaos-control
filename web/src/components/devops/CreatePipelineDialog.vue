<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref } from 'vue'
import * as jsYaml from 'js-yaml'
import YamlEditor from '@/components/common/YamlEditor.vue'
import { useDevOpsStore } from '@/stores/devops'
import { ApiError } from '@/api/client'

const props = defineProps<{
  open: boolean
  project: string
}>()

const emit = defineEmits<{
  close: []
  created: [pipeline: unknown]
}>()

const devops = useDevOpsStore()

const SKELETON = `name: My Pipeline
type: build

steps:
  - name: Step 1
    description: Describe what this step does
    command: echo "hello"
    timeout: 60s
`

const SLUG_RE = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/

const slug = ref('')
const definition = ref(SKELETON)
const submitting = ref(false)
const error = ref<string | null>(null)

function resetForm() {
  slug.value = ''
  definition.value = SKELETON
  submitting.value = false
  error.value = null
}

function handleClose() {
  resetForm()
  emit('close')
}

async function handleCreate() {
  error.value = null

  // Slug validation
  if (!slug.value || !SLUG_RE.test(slug.value)) {
    error.value = 'Slug must be lowercase alphanumeric with hyphens (e.g. my-pipeline).'
    return
  }

  // Client-side YAML validation
  try {
    jsYaml.load(definition.value)
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : 'Invalid YAML'
    error.value = `YAML parse error: ${msg}`
    return
  }

  submitting.value = true
  try {
    const pipeline = await devops.createPipeline(props.project, slug.value, definition.value)
    resetForm()
    emit('created', pipeline)
  } catch (e: unknown) {
    if (e instanceof ApiError) {
      if (e.status === 409) {
        error.value = 'A pipeline with this slug already exists.'
      } else {
        error.value = e.message
      }
    } else if (e instanceof Error) {
      error.value = e.message
    } else {
      error.value = 'Failed to create pipeline.'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div v-if="open" class="cpd-overlay" @click.self="handleClose">
    <div class="cpd-panel" role="dialog" aria-modal="true" aria-label="Create Pipeline">
      <h3 class="cpd-title">Create Pipeline</h3>

      <div class="cpd-field">
        <label class="cpd-label" for="pipeline-slug">Slug</label>
        <input
          id="pipeline-slug"
          v-model="slug"
          class="cpd-input"
          type="text"
          placeholder="my-pipeline"
          :disabled="submitting"
          autocomplete="off"
          spellcheck="false"
        />
        <span class="cpd-hint">Lowercase letters, numbers, and hyphens only.</span>
      </div>

      <div class="cpd-field cpd-field--editor">
        <label class="cpd-label">Pipeline Definition (YAML)</label>
        <div class="cpd-editor-wrap">
          <YamlEditor v-model="definition" :readonly="submitting" />
        </div>
      </div>

      <div v-if="error" class="cpd-error">{{ error }}</div>

      <div class="cpd-actions">
        <button
          class="btn-primary"
          :disabled="submitting"
          @click="handleCreate"
        >
          {{ submitting ? 'Creating…' : 'Create' }}
        </button>
        <button class="btn-ghost" :disabled="submitting" @click="handleClose">Cancel</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cpd-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
}

.cpd-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  padding: var(--space-6);
  width: 640px;
  max-width: 95vw;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.cpd-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.cpd-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.cpd-field--editor {
  flex: 1;
  min-height: 0;
}

.cpd-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
}

.cpd-input {
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  width: 100%;
  box-sizing: border-box;
}

.cpd-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.cpd-input:disabled {
  opacity: 0.6;
}

.cpd-hint {
  font-size: 11px;
  color: var(--color-text-muted);
}

.cpd-editor-wrap {
  height: 300px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.cpd-error {
  font-size: var(--text-sm);
  color: #dc2626;
  background: #fee2e2;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
}

.cpd-actions {
  display: flex;
  gap: var(--space-2);
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

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.88;
}

.btn-ghost {
  padding: var(--space-2) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}

.btn-ghost:hover:not(:disabled) {
  background: var(--color-surface);
}

.btn-ghost:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
