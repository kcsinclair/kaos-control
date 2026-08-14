<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ApiError } from '@/api/client'
import { nextAdrNumber, createAdr } from '@/api/architecture'
import MarkdownEditor from './MarkdownEditor.vue'

const props = defineProps<{ project: string }>()
const emit = defineEmits<{
  created: [path: string]
  close: []
}>()

const STATUS_OPTIONS = ['draft', 'approved', 'rejected', 'blocked'] as const

const title = ref('')
const slug = ref('')
const status = ref<typeof STATUS_OPTIONS[number]>('draft')
const body = ref('')
const slugEdited = ref(false)

const previewNumber = ref<number | null>(null)
const previewError = ref<string | null>(null)

const errors = ref<Record<string, string>>({})
const saving = ref(false)

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, ' ')
    .trim()
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
    .slice(0, 60)
}

function onTitleInput() {
  if (!slugEdited.value) slug.value = slugify(title.value)
}

function onSlugInput() {
  slugEdited.value = true
}

function paddedPreview(): string {
  return previewNumber.value !== null ? String(previewNumber.value).padStart(4, '0') : '????'
}

onMounted(async () => {
  try {
    previewNumber.value = await nextAdrNumber(props.project)
  } catch (e: unknown) {
    previewError.value = e instanceof Error ? e.message : 'Failed to preview next ADR number.'
  }
})

function validate(): boolean {
  const e: Record<string, string> = {}
  if (!title.value.trim()) e.title = 'Title is required.'
  if (!slug.value.trim()) e.slug = 'Slug is required.'
  errors.value = e
  return Object.keys(e).length === 0
}

async function submit() {
  if (saving.value) return
  if (!validate()) return
  saving.value = true
  try {
    const result = await createAdr(props.project, {
      slug: slug.value.trim(),
      title: title.value.trim(),
      status: status.value,
      body: body.value,
    })
    emit('created', result.path)
  } catch (e: unknown) {
    errors.value.submit = e instanceof ApiError ? e.message : e instanceof Error ? e.message : 'Failed to create ADR.'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="modal-overlay">
    <div class="modal-panel" role="dialog" aria-modal="true" aria-label="New ADR">
      <div class="modal-header">
        <h3 class="modal-title">New ADR</h3>
        <button class="btn-icon" aria-label="Close" @click="emit('close')">✕</button>
      </div>

      <form class="modal-body" @submit.prevent="submit">
        <p class="adr-preview">
          <template v-if="previewNumber !== null">This will create <strong>ADR-{{ paddedPreview() }}</strong></template>
          <template v-else-if="previewError">{{ previewError }}</template>
          <template v-else>Loading next ADR number…</template>
        </p>

        <div class="form-field">
          <label class="field-label" for="adr-title">Title</label>
          <input
            id="adr-title"
            v-model="title"
            class="field-input"
            :class="{ 'field-input--error': errors.title }"
            type="text"
            autocomplete="off"
            @input="onTitleInput"
          />
          <span v-if="errors.title" class="field-error">{{ errors.title }}</span>
        </div>

        <div class="form-field">
          <label class="field-label" for="adr-slug">Slug</label>
          <input
            id="adr-slug"
            v-model="slug"
            class="field-input mono"
            :class="{ 'field-input--error': errors.slug }"
            type="text"
            autocomplete="off"
            @input="onSlugInput"
          />
          <span v-if="errors.slug" class="field-error">{{ errors.slug }}</span>
        </div>

        <div class="form-field">
          <label class="field-label" for="adr-status">Status</label>
          <select id="adr-status" v-model="status" class="field-input field-select">
            <option v-for="s in STATUS_OPTIONS" :key="s" :value="s">{{ s }}</option>
          </select>
        </div>

        <div class="form-field form-field--body">
          <label class="field-label">Body</label>
          <div class="adr-body-editor">
            <MarkdownEditor v-model="body" />
          </div>
        </div>

        <span v-if="errors.submit" class="field-error">{{ errors.submit }}</span>
      </form>

      <div class="modal-footer">
        <button class="btn-primary" :disabled="saving" @click="submit">
          {{ saving ? 'Creating…' : 'Create' }}
        </button>
        <button class="btn-ghost" :disabled="saving" @click="emit('close')">Cancel</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
}
.modal-panel {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  width: 560px;
  max-width: calc(100vw - 2rem);
  max-height: calc(100vh - 4rem);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.modal-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.btn-icon {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  padding: var(--space-1);
  border-radius: var(--radius-sm);
  line-height: 1;
}
.btn-icon:hover { background: var(--color-surface); color: var(--color-text); }
.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-4) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.adr-preview {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.form-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.form-field--body {
  flex: 1;
  min-height: 0;
}
.field-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  flex-shrink: 0;
}
.field-input {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  width: 100%;
  box-sizing: border-box;
}
.field-input:focus { outline: none; border-color: var(--color-accent); }
.field-input--error { border-color: #dc2626; }
.field-select {
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M0 0l5 6 5-6z' fill='%23888'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right var(--space-2) center;
  padding-right: calc(var(--space-2) * 2 + 10px);
  cursor: pointer;
}
.field-error {
  font-size: 11px;
  color: #dc2626;
}
.mono { font-family: monospace; }
.adr-body-editor {
  height: 220px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.modal-footer {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-4) var(--space-6);
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
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
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary:hover:not(:disabled) { opacity: 0.88; }
.btn-ghost {
  padding: var(--space-2) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-ghost:hover { background: var(--color-surface); }
.btn-ghost:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
