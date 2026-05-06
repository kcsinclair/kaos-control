<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useSchedulerStore } from '@/stores/scheduler'
import { useUiStore } from '@/stores/ui'
import type { SchedulerJob, ScheduleSpec, Precondition } from '@/types/api'

const props = defineProps<{
  mode: 'create' | 'edit'
  project: string
  initial?: SchedulerJob
}>()

const emit = defineEmits<{
  (e: 'saved', job: SchedulerJob): void
  (e: 'cancel'): void
}>()

const store = useSchedulerStore()
const ui = useUiStore()

// ─── form state ──────────────────────────────────────────────────────────────
const name = ref(props.initial?.name ?? '')
const targetType = ref<'agent' | 'shell'>(props.initial?.target_type ?? 'shell')
const target = ref(props.initial?.target ?? '')
const scheduleType = ref<ScheduleSpec['type']>(props.initial?.schedule.type ?? 'cron')
const scheduleExpression = ref(props.initial?.schedule.expression ?? '')
const priority = ref(props.initial?.priority ?? 5)
const timeoutSec = ref(props.initial?.timeout_sec ?? 1800)
const enabled = ref(props.initial?.enabled ?? true)

interface PreconditionRow {
  type: Precondition['type']
  value: string
}

const preconditions = ref<PreconditionRow[]>(
  (props.initial?.preconditions ?? []).map((p) => ({ type: p.type, value: p.value })),
)

interface ArgRow {
  key: string
  value: string
}

const args = ref<ArgRow[]>(
  Object.entries(props.initial?.args ?? {}).map(([key, value]) => ({ key, value })),
)

// ─── validation ───────────────────────────────────────────────────────────────
const nameError = ref('')
const targetError = ref('')
const scheduleError = ref('')
const serverError = ref('')

const scheduleExpressPlaceholder = computed(() => {
  switch (scheduleType.value) {
    case 'cron':     return '0 2 * * *'
    case 'interval': return '30m'
    case 'once':     return '2026-06-01T02:00:00Z'
  }
})

const namePattern = /^[a-zA-Z0-9][a-zA-Z0-9-]{0,63}$/

function validateName(): boolean {
  if (!name.value) { nameError.value = 'Name is required'; return false }
  if (!namePattern.test(name.value)) {
    nameError.value = 'Name must be 1–64 alphanumeric + hyphens, starting with alphanumeric'
    return false
  }
  nameError.value = ''
  return true
}

function validateTarget(): boolean {
  if (!target.value) { targetError.value = 'Target is required'; return false }
  targetError.value = ''
  return true
}

function validateSchedule(): boolean {
  if (!scheduleExpression.value) { scheduleError.value = 'Schedule expression is required'; return false }
  scheduleError.value = ''
  return true
}

function validate(): boolean {
  const a = props.mode === 'create' ? validateName() : true
  const b = validateTarget()
  const c = validateSchedule()
  return a && b && c
}

// ─── preconditions helpers ────────────────────────────────────────────────────
function addPrecondition() {
  preconditions.value.push({ type: 'file_exists', value: '' })
}

function removePrecondition(i: number) {
  preconditions.value.splice(i, 1)
}

// ─── args helpers ─────────────────────────────────────────────────────────────
function addArg() {
  args.value.push({ key: '', value: '' })
}

function removeArg(i: number) {
  args.value.splice(i, 1)
}

// ─── reset validation on schedule type change ────────────────────────────────
watch(scheduleType, () => {
  scheduleExpression.value = ''
  scheduleError.value = ''
})

// ─── submit ───────────────────────────────────────────────────────────────────
const submitting = ref(false)

async function submit() {
  serverError.value = ''
  if (!validate()) return

  const argsMap = args.value.reduce<Record<string, string>>((acc, row) => {
    if (row.key) acc[row.key] = row.value
    return acc
  }, {})

  const payload = {
    name: name.value,
    target_type: targetType.value,
    target: target.value,
    args: Object.keys(argsMap).length ? argsMap : undefined,
    schedule: { type: scheduleType.value, expression: scheduleExpression.value },
    preconditions: preconditions.value.length ? preconditions.value : undefined,
    enabled: enabled.value,
    priority: priority.value,
    timeout_sec: timeoutSec.value,
  }

  submitting.value = true
  try {
    let saved: SchedulerJob
    if (props.mode === 'create') {
      saved = await store.createJob(props.project, payload)
      ui.success(`Job "${saved.name}" created`)
    } else {
      const { name: _n, ...update } = payload
      saved = await store.updateJob(props.project, props.initial!.name, update)
      ui.success(`Job "${saved.name}" updated`)
    }
    emit('saved', saved)
  } catch (e: unknown) {
    serverError.value = e instanceof Error ? e.message : 'An error occurred'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <form class="job-form" @submit.prevent="submit">
    <div class="form-body">

      <!-- Name (create only) -->
      <div v-if="mode === 'create'" class="form-field">
        <label class="field-label" for="jf-name">Name</label>
        <input
          id="jf-name"
          v-model="name"
          type="text"
          class="field-input"
          :class="{ 'field-input--error': nameError }"
          placeholder="my-daily-backup"
          @blur="validateName"
        />
        <p v-if="nameError" class="field-error">{{ nameError }}</p>
      </div>

      <!-- Target type -->
      <div class="form-field">
        <fieldset class="field-fieldset">
          <legend class="field-label">Target type</legend>
          <label class="radio-label">
            <input v-model="targetType" type="radio" value="shell" /> Shell
          </label>
          <label class="radio-label">
            <input v-model="targetType" type="radio" value="agent" /> Agent
          </label>
        </fieldset>
      </div>

      <!-- Target -->
      <div class="form-field">
        <label class="field-label" for="jf-target">Target</label>
        <input
          id="jf-target"
          v-model="target"
          type="text"
          class="field-input"
          :class="{ 'field-input--error': targetError }"
          :placeholder="targetType === 'shell' ? '/scripts/backup.sh' : 'backend-developer'"
          @blur="validateTarget"
        />
        <p v-if="targetError" class="field-error">{{ targetError }}</p>
      </div>

      <!-- Schedule type -->
      <div class="form-field">
        <fieldset class="field-fieldset">
          <legend class="field-label">Schedule type</legend>
          <label class="radio-label">
            <input v-model="scheduleType" type="radio" value="cron" /> Cron
          </label>
          <label class="radio-label">
            <input v-model="scheduleType" type="radio" value="interval" /> Interval
          </label>
          <label class="radio-label">
            <input v-model="scheduleType" type="radio" value="once" /> Once
          </label>
        </fieldset>
      </div>

      <!-- Schedule expression -->
      <div class="form-field">
        <label class="field-label" for="jf-schedule">Schedule expression</label>
        <input
          id="jf-schedule"
          v-model="scheduleExpression"
          type="text"
          class="field-input field-input--mono"
          :class="{ 'field-input--error': scheduleError }"
          :placeholder="scheduleExpressPlaceholder"
          @blur="validateSchedule"
        />
        <p v-if="scheduleError" class="field-error">{{ scheduleError }}</p>
      </div>

      <!-- Priority -->
      <div class="form-field form-field--row">
        <label class="field-label" for="jf-priority">Priority (1–10)</label>
        <div class="priority-row">
          <input
            id="jf-priority"
            v-model.number="priority"
            type="range"
            min="1"
            max="10"
            class="priority-range"
          />
          <span class="priority-value">{{ priority }}</span>
        </div>
      </div>

      <!-- Timeout -->
      <div class="form-field">
        <label class="field-label" for="jf-timeout">Timeout (seconds)</label>
        <input
          id="jf-timeout"
          v-model.number="timeoutSec"
          type="number"
          min="1"
          class="field-input field-input--sm"
          placeholder="1800"
        />
      </div>

      <!-- Enabled -->
      <div class="form-field form-field--row">
        <label class="field-label" for="jf-enabled">Enabled</label>
        <input id="jf-enabled" v-model="enabled" type="checkbox" class="field-checkbox" />
      </div>

      <!-- Preconditions -->
      <div class="form-field">
        <div class="field-group-header">
          <span class="field-label">Preconditions</span>
          <button type="button" class="btn-add" @click="addPrecondition">+ Add</button>
        </div>
        <div class="precond-list">
          <div
            v-for="(pc, i) in preconditions"
            :key="i"
            class="precond-row"
          >
            <select v-model="pc.type" class="precond-type-select">
              <option value="after_job">After job</option>
              <option value="file_exists">File exists</option>
              <option value="http_ok">HTTP OK</option>
              <option value="shell">Shell</option>
            </select>
            <input
              v-model="pc.value"
              type="text"
              class="precond-value-input"
              placeholder="value"
            />
            <button type="button" class="btn-remove" @click="removePrecondition(i)">✕</button>
          </div>
          <p v-if="!preconditions.length" class="field-hint">No preconditions — job runs unconditionally.</p>
        </div>
      </div>

      <!-- Args -->
      <div class="form-field">
        <div class="field-group-header">
          <span class="field-label">Args (key-value)</span>
          <button type="button" class="btn-add" @click="addArg">+ Add</button>
        </div>
        <div class="args-list">
          <div v-for="(arg, i) in args" :key="i" class="arg-row">
            <input v-model="arg.key"   type="text" class="arg-key-input"   placeholder="key" />
            <span class="arg-sep">=</span>
            <input v-model="arg.value" type="text" class="arg-value-input" placeholder="value" />
            <button type="button" class="btn-remove" @click="removeArg(i)">✕</button>
          </div>
          <p v-if="!args.length" class="field-hint">No args.</p>
        </div>
      </div>

      <!-- Server error -->
      <p v-if="serverError" class="server-error">{{ serverError }}</p>
    </div>

    <!-- Footer -->
    <div class="form-footer">
      <button type="button" class="btn-secondary" @click="emit('cancel')">Cancel</button>
      <button type="submit" class="btn-primary" :disabled="submitting">
        {{ submitting ? 'Saving…' : (mode === 'create' ? 'Create Job' : 'Save Changes') }}
      </button>
    </div>
  </form>
</template>

<style scoped>
.job-form {
  display: flex;
  flex-direction: column;
}

.form-body {
  padding: var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  flex: 1;
}

.form-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-6);
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.form-field--row {
  flex-direction: row;
  align-items: center;
  gap: var(--space-3);
}

.field-label {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}

.field-input {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  background: var(--color-bg);
  color: var(--color-text);
}
.field-input--error { border-color: var(--color-error); }
.field-input--mono  { font-family: monospace; }
.field-input--sm    { max-width: 120px; }

.field-error {
  font-size: var(--text-xs);
  color: var(--color-error);
  margin: 0;
}

.field-hint {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
  margin: 0;
}

.field-fieldset {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  padding: var(--space-2) var(--space-3);
  display: flex;
  gap: var(--space-4);
}
.field-fieldset legend { font-size: var(--text-sm); font-weight: 500; color: var(--color-text); }

.radio-label {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--text-sm);
  cursor: pointer;
  color: var(--color-text);
}

.field-checkbox { width: 16px; height: 16px; cursor: pointer; }

.priority-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.priority-range { flex: 1; }
.priority-value {
  font-size: var(--text-sm);
  font-weight: 600;
  min-width: 20px;
  text-align: center;
  color: var(--color-text);
}

.field-group-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.btn-add {
  font-size: var(--text-xs);
  padding: 2px var(--space-2);
  border: 1px solid var(--color-accent);
  border-radius: var(--radius-sm);
  background: none;
  color: var(--color-accent);
  cursor: pointer;
}
.btn-add:hover { background: var(--color-accent-subtle); }

.btn-remove {
  padding: 1px 6px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: none;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: 11px;
}
.btn-remove:hover { border-color: var(--color-error); color: var(--color-error); }

.precond-list,
.args-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.precond-row,
.arg-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.precond-type-select {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  background: var(--color-bg);
  color: var(--color-text);
  flex-shrink: 0;
  width: 130px;
}

.precond-value-input,
.arg-value-input {
  flex: 1;
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  background: var(--color-bg);
  color: var(--color-text);
}

.arg-key-input {
  width: 120px;
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  background: var(--color-bg);
  color: var(--color-text);
  flex-shrink: 0;
}
.arg-sep { color: var(--color-text-muted); font-weight: 600; }

.server-error {
  font-size: var(--text-sm);
  color: var(--color-error);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-error);
  border-radius: var(--radius-sm);
  background: var(--badge-blocked-bg);
  margin: 0;
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
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-secondary {
  padding: var(--space-2) var(--space-4);
  background: none;
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  cursor: pointer;
}
.btn-secondary:hover { background: var(--color-bg); color: var(--color-text); }
</style>
