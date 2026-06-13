<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, watch } from 'vue'
import * as jsYaml from 'js-yaml'
import YamlEditor from '@/components/common/YamlEditor.vue'
import { useDevOpsStore } from '@/stores/devops'
import * as devopsApi from '@/api/devops'
import { ApiError } from '@/api/client'

const props = defineProps<{
  open: boolean
  project: string
  slug: string
}>()

const emit = defineEmits<{
  close: []
  updated: []
}>()

const devops = useDevOpsStore()

const definition = ref('')
const originalDefinition = ref('')
const loading = ref(false)
const submitting = ref(false)
const error = ref<string | null>(null)
const confirmRemove = ref(false)
const removedStepCount = ref(0)

function countSteps(yaml: string): number {
  try {
    const parsed = jsYaml.load(yaml) as Record<string, unknown> | null
    if (!parsed || !Array.isArray(parsed['steps'])) return 0
    return (parsed['steps'] as unknown[]).length
  } catch {
    return 0
  }
}

function validateYaml(yaml: string): string | null {
  let parsed: unknown
  try {
    parsed = jsYaml.load(yaml)
  } catch (e: unknown) {
    return `YAML parse error: ${e instanceof Error ? e.message : 'Invalid YAML'}`
  }
  if (!parsed || typeof parsed !== 'object') return 'Definition must be a YAML mapping.'
  const obj = parsed as Record<string, unknown>
  if (!obj['name']) return 'Missing required field: name'
  if (!obj['type']) return 'Missing required field: type'
  if (!Array.isArray(obj['steps']) || obj['steps'].length === 0) {
    return 'Missing required field: steps (must be a non-empty list)'
  }
  const steps = obj['steps'] as unknown[]
  for (let i = 0; i < steps.length; i++) {
    const step = steps[i] as Record<string, unknown>
    if (!step || !step['name'] || !step['command']) {
      return `Step ${i + 1} is missing required fields: name and command`
    }
  }
  return null
}

async function loadDefinition() {
  if (!props.slug) return
  loading.value = true
  error.value = null
  try {
    const yaml = await devopsApi.getPipelineDefinition(props.project, props.slug)
    definition.value = yaml
    originalDefinition.value = yaml
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load pipeline definition.'
  } finally {
    loading.value = false
  }
}

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      confirmRemove.value = false
      removedStepCount.value = 0
      error.value = null
      loadDefinition()
    }
  },
)

function handleClose() {
  confirmRemove.value = false
  emit('close')
}

const isDirty = (): boolean => definition.value !== originalDefinition.value

const isValid = (): boolean => isDirty() && validateYaml(definition.value) === null

async function handleSave() {
  error.value = null

  const validationError = validateYaml(definition.value)
  if (validationError) {
    error.value = validationError
    return
  }

  if (!isDirty()) return

  // Check for step removal
  if (!confirmRemove.value) {
    const oldCount = countSteps(originalDefinition.value)
    const newCount = countSteps(definition.value)
    if (newCount < oldCount) {
      removedStepCount.value = oldCount - newCount
      confirmRemove.value = true
      return
    }
  }

  confirmRemove.value = false
  submitting.value = true
  try {
    await devops.updatePipeline(props.project, props.slug, definition.value)
    emit('updated')
    emit('close')
  } catch (e: unknown) {
    if (e instanceof ApiError) {
      if (e.status === 409) {
        error.value = 'Cannot save: a pipeline run is currently active. Wait for it to finish.'
      } else {
        error.value = e.message
      }
    } else if (e instanceof Error) {
      error.value = e.message
    } else {
      error.value = 'Failed to save pipeline.'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div v-if="open" class="epd-overlay">
    <div class="epd-panel" role="dialog" aria-modal="true" aria-label="Edit Pipeline">
      <h3 class="epd-title">Edit Pipeline</h3>

      <div class="epd-field">
        <span class="epd-label">Slug</span>
        <span class="epd-slug">{{ slug }}</span>
      </div>

      <div v-if="loading" class="epd-loading">Loading definition…</div>

      <template v-else>
        <div class="epd-field epd-field--editor">
          <label class="epd-label">Pipeline Definition (YAML)</label>
          <div class="epd-editor-wrap">
            <YamlEditor v-model="definition" :readonly="submitting" />
          </div>
        </div>

        <div v-if="confirmRemove" class="epd-confirm">
          You are removing {{ removedStepCount }} step{{ removedStepCount !== 1 ? 's' : '' }}. Save anyway?
          <div class="epd-confirm-actions">
            <button class="btn-primary" :disabled="submitting" @click="handleSave">
              {{ submitting ? 'Saving…' : 'Yes, save' }}
            </button>
            <button class="btn-ghost" :disabled="submitting" @click="confirmRemove = false">Cancel</button>
          </div>
        </div>

        <div v-if="error" class="epd-error">{{ error }}</div>

        <div v-if="!confirmRemove" class="epd-actions">
          <button
            class="btn-primary"
            :disabled="submitting || !isValid()"
            @click="handleSave"
          >
            {{ submitting ? 'Saving…' : 'Save' }}
          </button>
          <button class="btn-ghost" :disabled="submitting" @click="handleClose">Cancel</button>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.epd-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
}

.epd-panel {
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

.epd-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.epd-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.epd-field--editor {
  flex: 1;
  min-height: 0;
}

.epd-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
}

.epd-slug {
  font-size: var(--text-sm);
  font-family: monospace;
  color: var(--color-text);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: var(--space-2);
}

.epd-loading {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  padding: var(--space-4) 0;
  text-align: center;
}

.epd-editor-wrap {
  height: 300px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.epd-error {
  font-size: var(--text-sm);
  color: #dc2626;
  background: #fee2e2;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
}

.epd-confirm {
  font-size: var(--text-sm);
  color: var(--color-text);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.epd-confirm-actions {
  display: flex;
  gap: var(--space-2);
}

.epd-actions {
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
