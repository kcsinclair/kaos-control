<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
// Milestone 7 (FR-17, FR-18): a final, opt-in step offering
// config/pipelines/agent-directives/repo-skeleton scaffolding. Never
// automatic — nothing runs until "Run scaffolding" is clicked. Each step's
// naming fields default to the backend-supplied defaults; a "decide for
// me" control resets that step back to those defaults (the backend's
// ScaffoldChoice is one use_defaults flag per step, not per field).
import { computed, onMounted, reactive, ref } from 'vue'
import { getScaffold, runScaffold } from '@/api/architecture'
import { ApiError } from '@/api/client'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import type { ScaffoldChoice, ScaffoldResult, WizardScaffoldAvailability } from '@/types/api'

const props = defineProps<{
  project: string
}>()

const store = useArchitectureWizardStore()

const availability = ref<WizardScaffoldAvailability | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const running = ref(false)
const result = ref<ScaffoldResult | null>(null)

const stepState = reactive<Record<string, { useDefaults: boolean; values: Record<string, string> }>>({})

function defaultsFor(fields: { key: string; default_value: string }[] | undefined): Record<string, string> {
  return Object.fromEntries((fields ?? []).map((f) => [f.key, f.default_value]))
}

async function load(): Promise<void> {
  const arch = store.chosenArchitecture
  const stack = store.chosenStack
  if (!arch || !stack) return
  loading.value = true
  error.value = null
  try {
    availability.value = await getScaffold(props.project, arch.slug, stack.slug)
    for (const s of availability.value.steps ?? []) {
      stepState[s.key] = { useDefaults: true, values: defaultsFor(s.name_fields) }
    }
  } catch (e: unknown) {
    error.value = e instanceof ApiError ? e.message : 'Failed to load scaffolding options.'
  } finally {
    loading.value = false
  }
}

function onFieldInput(stepKey: string, fieldKey: string, value: string): void {
  stepState[stepKey].values[fieldKey] = value
  stepState[stepKey].useDefaults = false
}

function decideForMe(stepKey: string): void {
  const step = availability.value?.steps?.find((s) => s.key === stepKey)
  if (!step) return
  stepState[stepKey] = { useDefaults: true, values: defaultsFor(step.name_fields) }
}

const choices = computed<ScaffoldChoice[]>(() =>
  (availability.value?.steps ?? []).map((s) => ({
    step_key: s.key,
    values: stepState[s.key]?.values,
    use_defaults: stepState[s.key]?.useDefaults ?? true,
  })),
)

async function submit(): Promise<void> {
  const arch = store.chosenArchitecture
  const stack = store.chosenStack
  if (!arch || !stack) return
  running.value = true
  error.value = null
  try {
    const res = await runScaffold(props.project, arch.slug, stack.slug, choices.value)
    result.value = res.result ?? null
  } catch (e: unknown) {
    error.value = e instanceof ApiError ? e.message : 'Failed to run scaffolding.'
  } finally {
    running.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="scaffold-step">
    <h2 class="scaffold-title">Optional scaffolding</h2>

    <div v-if="loading" class="scaffold-state" role="status" aria-live="polite">
      Checking what scaffolding is available…
    </div>
    <div v-else-if="error" class="scaffold-state error" role="alert">{{ error }}</div>

    <div v-else-if="!availability?.available" class="scaffold-state not-available">
      {{ availability?.message || 'Scaffolding isn\'t available for this stack yet.' }}
      See [[agent-directives-generation]] — coming soon.
    </div>

    <template v-else-if="result">
      <div class="scaffold-result" role="status">
        <p v-if="result.applied.length">Applied:</p>
        <ul>
          <li v-for="p in result.applied" :key="p">{{ p }}</li>
        </ul>
        <p v-if="result.skipped.length">Skipped (already present):</p>
        <ul>
          <li v-for="p in result.skipped" :key="p">{{ p }}</li>
        </ul>

        <p v-if="result.committed" class="scaffold-committed">✓ Committed to git.</p>
        <div v-else-if="result.git_commands?.length" class="scaffold-git">
          <p>These files aren't committed yet — run these to track them:</p>
          <pre class="scaffold-git-cmds">{{ result.git_commands.join('\n') }}</pre>
        </div>
      </div>
    </template>

    <template v-else>
      <div v-for="s in availability.steps" :key="s.key" class="scaffold-step-card">
        <h3 class="scaffold-step-title">{{ s.title }}</h3>
        <p class="scaffold-step-desc">{{ s.description }}</p>

        <div v-if="s.name_fields?.length" class="scaffold-fields">
          <label v-for="f in s.name_fields" :key="f.key" class="scaffold-field">
            {{ f.label }}
            <input
              type="text"
              :value="stepState[s.key]?.values[f.key]"
              @input="onFieldInput(s.key, f.key, ($event.target as HTMLInputElement).value)"
            />
          </label>
          <button type="button" class="decide-btn" @click="decideForMe(s.key)">
            Not sure — decide for me
          </button>
        </div>
      </div>

      <button
        type="button"
        class="btn-primary run-scaffold-btn"
        :disabled="running"
        @click="submit"
      >{{ running ? 'Running…' : 'Run scaffolding' }}</button>
    </template>
  </div>
</template>

<style scoped>
.scaffold-step {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.scaffold-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
}
.scaffold-state {
  padding: var(--space-4);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.scaffold-state.error { color: #991b1b; }
.scaffold-state.not-available {
  background: var(--color-surface);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-md);
  line-height: 1.6;
}
.scaffold-step-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}
.scaffold-step-title {
  margin: 0;
  font-size: var(--text-base);
  font-weight: 700;
  color: var(--color-text);
}
.scaffold-step-desc {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.scaffold-fields {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  align-items: flex-start;
}
.scaffold-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  font-size: var(--text-sm);
  color: var(--color-text);
  width: 100%;
}
.scaffold-field input {
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg, #fff);
  color: var(--color-text);
}
.decide-btn {
  background: none;
  border: none;
  padding: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
  text-decoration: underline;
}
.scaffold-result ul {
  margin: 0 0 var(--space-2);
  padding-left: 1.2em;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.scaffold-committed {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-accent);
}
.scaffold-git {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.scaffold-git-cmds {
  margin: var(--space-2) 0 0;
  padding: var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--text-xs);
  overflow-x: auto;
  white-space: pre;
}
.run-scaffold-btn {
  align-self: flex-start;
}
.btn-primary {
  padding: var(--space-2) var(--space-5);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-primary:not(:disabled):hover { opacity: 0.88; }
</style>
