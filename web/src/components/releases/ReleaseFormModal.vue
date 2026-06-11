<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useReleasesStore } from '@/stores/releases'
import { ApiError } from '@/api/client'
import type { Release, ReleaseStatus } from '@/types/release'

const props = defineProps<{
  release?: Release
  project: string
}>()

const emit = defineEmits<{
  saved: [release: Release]
  close: []
}>()

const router = useRouter()
const store = useReleasesStore()

const isEdit = computed(() => props.release !== undefined)

const name = ref('')
const status = ref<ReleaseStatus>('planned')
const isScheduled = ref(true)
const startDate = ref('')
const durationValue = ref<number>(7)
const durationUnit = ref<'days' | 'weeks'>('days')
const endDate = ref('')

// Which input was last changed — used to decide which to auto-recalculate
const lastChanged = ref<'duration' | 'end'>('end')

const errors = ref<Record<string, string>>({})
const saving = ref(false)

function formatDateForInput(dateStr: string | null | undefined): string {
  if (!dateStr) return ''
  // Already in YYYY-MM-DD or ISO format
  return dateStr.slice(0, 10)
}

function calcEndFromDuration(): string {
  if (!startDate.value) return ''
  const start = new Date(startDate.value)
  const days = durationUnit.value === 'weeks' ? durationValue.value * 7 : durationValue.value
  start.setDate(start.getDate() + days - 1)
  return start.toISOString().slice(0, 10)
}

function calcDurationFromEnd(): void {
  if (!startDate.value || !endDate.value) return
  const start = new Date(startDate.value)
  const end = new Date(endDate.value)
  const diffMs = end.getTime() - start.getTime()
  const diffDays = Math.round(diffMs / 86400000) + 1
  if (durationUnit.value === 'weeks') {
    durationValue.value = Math.max(1, Math.round(diffDays / 7))
  } else {
    durationValue.value = Math.max(1, diffDays)
  }
}

function onStartDateChange() {
  if (lastChanged.value === 'duration') {
    endDate.value = calcEndFromDuration()
  } else {
    calcDurationFromEnd()
  }
}

function onDurationChange() {
  lastChanged.value = 'duration'
  endDate.value = calcEndFromDuration()
}

function onEndDateChange() {
  lastChanged.value = 'end'
  calcDurationFromEnd()
}

function onDurationUnitChange() {
  if (lastChanged.value === 'duration') {
    endDate.value = calcEndFromDuration()
  } else {
    calcDurationFromEnd()
  }
}

function onStatusChange() {
  if (status.value === 'unscheduled') {
    isScheduled.value = false
  }
}

function setScheduled(val: boolean) {
  isScheduled.value = val
  if (val && status.value === 'unscheduled') {
    status.value = 'planned'
  }
}

function navigateToFile() {
  if (props.release?.file_path) {
    router.push(`/p/${encodeURIComponent(props.project)}/artifacts/${props.release.file_path}`)
    emit('close')
  }
}

onMounted(() => {
  if (props.release) {
    name.value = props.release.name
    status.value = props.release.status
    if (props.release.start_date && props.release.end_date) {
      isScheduled.value = true
      startDate.value = formatDateForInput(props.release.start_date)
      endDate.value = formatDateForInput(props.release.end_date)
      calcDurationFromEnd()
    } else {
      isScheduled.value = false
    }
  }
})

function validate(): boolean {
  const e: Record<string, string> = {}
  const trimmedName = name.value.trim()
  if (!trimmedName) {
    e.name = 'Name is required.'
  } else if (name.value.length > 120) {
    e.name = 'Name must be 120 characters or fewer.'
  } else if (!isEdit.value && store.byName(trimmedName)) {
    // Pre-flight: catch conflicts before any network request is sent.
    e.name = `A release named "${trimmedName}" already exists.`
  } else if (
    isEdit.value &&
    props.release &&
    trimmedName !== props.release.name &&
    store.byName(trimmedName)
  ) {
    e.name = `A release named "${trimmedName}" already exists.`
  }
  if (isScheduled.value) {
    if (!startDate.value) {
      e.startDate = 'Start date is required when scheduled.'
    }
    if (!endDate.value) {
      e.endDate = 'End date is required when scheduled.'
    } else if (startDate.value && endDate.value < startDate.value) {
      e.endDate = 'End date must be on or after start date.'
    }
  }
  errors.value = e
  return Object.keys(e).length === 0
}

async function submit() {
  // Guard: prevent concurrent submissions (e.g. rapid double-clicks or
  // pressing Enter while the first request is still in flight).
  if (saving.value) return
  if (!validate()) return
  saving.value = true
  try {
    const payload = {
      name: name.value.trim(),
      status: status.value,
      start_date: isScheduled.value ? startDate.value : null,
      end_date: isScheduled.value ? endDate.value : null,
    }
    let release: Release
    if (isEdit.value && props.release) {
      release = await store.update(props.project, props.release.id, payload)
    } else {
      release = await store.create(props.project, payload)
    }
    emit('saved', release)
  } catch (e: unknown) {
    if (e instanceof ApiError && e.status === 409) {
      if (e.message.includes('already in use') || e.message.includes('already exists')) {
        errors.value.name = 'A release with this name already exists.'
      } else {
        errors.value.submit = 'This release was changed by another session — reload to continue.'
      }
    } else {
      errors.value.submit = e instanceof Error ? e.message : 'Save failed.'
    }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal-panel" role="dialog" aria-modal="true" :aria-label="isEdit ? 'Edit release' : 'Create release'">
      <div class="modal-header">
        <h3 class="modal-title">{{ isEdit ? 'Edit Release' : 'New Release' }}</h3>
        <button class="btn-icon" aria-label="Close" @click="emit('close')">✕</button>
      </div>

      <form class="modal-body" @submit.prevent="submit">
        <div class="form-field">
          <label class="field-label" for="rel-name">Name</label>
          <input
            id="rel-name"
            v-model="name"
            class="field-input"
            :class="{ 'field-input--error': errors.name }"
            type="text"
            maxlength="120"
            placeholder="e.g. v1.0"
            autocomplete="off"
          />
          <span v-if="errors.name" class="field-error">{{ errors.name }}</span>
        </div>

        <div v-if="isEdit && release?.file_path" class="form-field">
          <span class="field-label">File</span>
          <button type="button" class="file-path-chip" @click="navigateToFile">
            {{ release.file_path }}
          </button>
        </div>

        <div class="form-field">
          <label class="field-label" for="rel-status">Status</label>
          <select id="rel-status" v-model="status" class="field-input field-select" @change="onStatusChange">
            <option value="planned">planned</option>
            <option value="active">active</option>
            <option value="shipped">shipped</option>
            <option value="unscheduled">unscheduled</option>
          </select>
        </div>

        <div class="form-field form-field--row">
          <span class="field-label">Schedule</span>
          <div class="toggle-group" role="group" aria-label="Schedule toggle">
            <button
              type="button"
              class="toggle-btn"
              :class="{ 'toggle-btn--active': isScheduled }"
              @click="setScheduled(true)"
            >Scheduled</button>
            <button
              type="button"
              class="toggle-btn"
              :class="{ 'toggle-btn--active': !isScheduled }"
              @click="setScheduled(false)"
            >Unscheduled</button>
          </div>
        </div>

        <template v-if="isScheduled">
          <div class="form-field">
            <label class="field-label" for="rel-start">Start Date</label>
            <input
              id="rel-start"
              v-model="startDate"
              class="field-input"
              :class="{ 'field-input--error': errors.startDate }"
              type="date"
              @change="onStartDateChange"
            />
            <span v-if="errors.startDate" class="field-error">{{ errors.startDate }}</span>
          </div>

          <div class="form-field">
            <label class="field-label">Duration</label>
            <div class="duration-row">
              <input
                v-model.number="durationValue"
                class="field-input duration-num"
                type="number"
                min="1"
                @input="onDurationChange"
              />
              <select
                v-model="durationUnit"
                class="field-input field-select duration-unit"
                @change="onDurationUnitChange"
              >
                <option value="days">days</option>
                <option value="weeks">weeks</option>
              </select>
            </div>
          </div>

          <div class="form-field">
            <label class="field-label" for="rel-end">End Date</label>
            <input
              id="rel-end"
              v-model="endDate"
              class="field-input"
              :class="{ 'field-input--error': errors.endDate }"
              type="date"
              :min="startDate"
              @change="onEndDateChange"
            />
            <span v-if="errors.endDate" class="field-error">{{ errors.endDate }}</span>
          </div>
        </template>

        <span v-if="errors.submit" class="field-error">{{ errors.submit }}</span>
      </form>

      <div class="modal-footer">
        <button class="btn-primary" :disabled="saving" @click="submit">
          {{ saving ? 'Saving…' : isEdit ? 'Save Changes' : 'Create' }}
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
  width: 460px;
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
.form-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.form-field--row {
  flex-direction: row;
  align-items: center;
  gap: var(--space-3);
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
.toggle-group {
  display: flex;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.toggle-btn {
  padding: var(--space-1) var(--space-3);
  background: none;
  border: none;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}
.toggle-btn + .toggle-btn {
  border-left: 1px solid var(--color-border);
}
.toggle-btn--active {
  background: var(--color-accent);
  color: #fff;
}
.toggle-btn:hover:not(.toggle-btn--active) {
  background: var(--color-surface);
  color: var(--color-text);
}
.duration-row {
  display: flex;
  gap: var(--space-2);
}
.duration-num {
  width: 80px;
}
.duration-unit {
  flex: 1;
}
.file-path-chip {
  display: inline-block;
  padding: 2px var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-accent);
  font-size: 11px;
  font-family: monospace;
  cursor: pointer;
  text-align: left;
  text-decoration: underline;
  text-decoration-style: dotted;
}
.file-path-chip:hover {
  background: var(--color-bg);
  text-decoration-style: solid;
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
