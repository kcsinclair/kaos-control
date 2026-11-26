<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useProvidersStore } from '@/stores/providers'
import type { AgentSummary } from '@/types/api'

// The shape the form emits — mirrors the YAML agent config fields.
export interface AgentFormData {
  name: string
  roles: string[]
  driver: 'claude-code-cli' | 'claude-mediated' | 'codex-cli' | 'gemini' | 'gemini-cli' | 'openai-compatible'
  model: string
  provider?: string
  max_tool_iterations?: number
  allowed_write_paths: string[]
  timeout_minutes: number
  git_identity_name: string
  git_identity_email: string
  prompt_templates: Record<string, string>
  fallback_provider?: string
  fallback_model?: string
}

const props = defineProps<{
  initial?: AgentSummary | null
  availableRoles: string[]
  existingNames?: string[]
}>()

const emit = defineEmits<{
  submit: [data: AgentFormData]
  cancel: []
}>()

const providersStore = useProvidersStore()

const isEdit = !!props.initial

// ── Form state ─────────────────────────────────────────────────────────────
const name = ref(props.initial?.name ?? '')
const selectedRoles = ref<string[]>(props.initial?.roles ?? [])
type DriverChoice = 'claude-code-cli' | 'claude-mediated' | 'codex-cli' | 'gemini' | 'gemini-cli' | 'openai-compatible'
const driver = ref<DriverChoice>(
  (props.initial?.driver && props.initial.driver !== 'ollama' ? props.initial.driver : 'claude-code-cli') as DriverChoice,
)
const model = ref(props.initial?.model ?? '')
const provider = ref(props.initial?.provider ?? '')
const maxToolIterations = ref<number | undefined>(props.initial?.max_tool_iterations)
const allowedWritePathsRaw = ref((props.initial?.allowed_write_paths ?? []).join('\n'))
const timeoutMinutes = ref(props.initial?.timeout_minutes ?? 0)
const gitIdentityName = ref(props.initial?.git_identity?.name ?? '')
const gitIdentityEmail = ref(props.initial?.git_identity?.email ?? '')
const fallbackProvider = ref(props.initial?.fallback_provider ?? '')
const fallbackModel = ref(props.initial?.fallback_model ?? '')

// Prompt templates: one labelled textarea per role, so multi-role template
// maps (e.g. idea-capture's 3 templates) round-trip losslessly instead of
// collapsing through a single delimited textarea.
const promptTemplates = ref<Record<string, string>>({ ...(props.initial?.prompt_templates ?? {}) })
const promptTemplateRoles = ref<string[]>(Object.keys(promptTemplates.value))
const newTemplateRole = ref('')
const templateRolesAvailableToAdd = computed(() =>
  props.availableRoles.filter((r) => !promptTemplateRoles.value.includes(r)),
)

function addTemplateRole() {
  const role = newTemplateRole.value
  if (!role || promptTemplateRoles.value.includes(role)) return
  promptTemplateRoles.value.push(role)
  promptTemplates.value[role] = ''
  newTemplateRole.value = ''
}

function removeTemplateRole(role: string) {
  promptTemplateRoles.value = promptTemplateRoles.value.filter((r) => r !== role)
  delete promptTemplates.value[role]
}

const errors = ref<Record<string, string>>({})

// ── Provider model list ───────────────────────────────────────────────────
const providerModels = computed(() => {
  if (!provider.value) return []
  return providersStore.models.get(provider.value) ?? []
})

const providerHealth = computed(() => {
  if (!provider.value) return null
  return providersStore.health.get(provider.value) ?? providersStore.probeResults.get(provider.value) ?? null
})

const fetchingProviderModels = ref(false)

async function loadProviderModels() {
  if (!provider.value) return
  fetchingProviderModels.value = true
  try {
    await providersStore.fetchModels(provider.value)
  } finally {
    fetchingProviderModels.value = false
  }
}

watch(provider, (val) => {
  if (val) void loadProviderModels()
  if (driver.value === 'openai-compatible' && !isEdit) {
    model.value = ''
  }
})

onMounted(async () => {
  try {
    if (!providersStore.providers.length) {
      await providersStore.fetchProviders().catch(() => {})
    }
    if (provider.value) {
      await loadProviderModels().catch(() => {})
    }
  } catch {
    // Non-fatal if stores fail to load on mount in test environment
  }
})

// ── Validation ─────────────────────────────────────────────────────────────
function validate(): boolean {
  const e: Record<string, string> = {}
  if (!name.value.trim()) e.name = 'Name is required.'
  else if (!isEdit && props.existingNames?.includes(name.value.trim()))
    e.name = 'An agent with this name already exists.'
  if (!selectedRoles.value.length) e.roles = 'At least one role is required.'
  if (driver.value === 'openai-compatible') {
    if (!provider.value) e.provider = 'Select a provider.'
    if (!model.value.trim()) e.model = 'Model is required for OpenAI-compatible driver.'
  } else if (
    driver.value === 'claude-code-cli' ||
    driver.value === 'claude-mediated' ||
    driver.value === 'gemini'
  ) {
    // codex-cli and gemini-cli use the binary's default model when none is given.
    if (!model.value.trim()) e.model = 'Model is required.'
  }
  if (fallbackProvider.value) {
    if (fallbackProvider.value === (driver.value === 'openai-compatible' ? provider.value : undefined))
      e.fallbackProvider = 'Fallback provider must differ from the primary provider.'
    if (!fallbackModel.value.trim()) e.fallbackModel = 'Fallback model is required when a fallback provider is set.'
  }
  errors.value = e
  return Object.keys(e).length === 0
}

function handleSubmit() {
  if (!validate()) return
  emit('submit', {
    name: name.value.trim(),
    roles: selectedRoles.value,
    driver: driver.value,
    model: model.value.trim(),
    provider: driver.value === 'openai-compatible' ? provider.value : undefined,
    max_tool_iterations: maxToolIterations.value && !isNaN(maxToolIterations.value) && maxToolIterations.value > 0 ? maxToolIterations.value : undefined,
    allowed_write_paths: allowedWritePathsRaw.value
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean),
    timeout_minutes: timeoutMinutes.value,
    git_identity_name: gitIdentityName.value.trim(),
    git_identity_email: gitIdentityEmail.value.trim(),
    fallback_provider: fallbackProvider.value || undefined,
    fallback_model: fallbackProvider.value ? fallbackModel.value.trim() : undefined,
    prompt_templates: collectPromptTemplates(),
  })
}

function collectPromptTemplates(): Record<string, string> {
  const result: Record<string, string> = {}
  for (const role of promptTemplateRoles.value) {
    const body = promptTemplates.value[role] ?? ''
    if (body.trim() === '') continue
    result[role] = body
  }
  return result
}

function toggleRole(role: string) {
  const idx = selectedRoles.value.indexOf(role)
  if (idx >= 0) selectedRoles.value.splice(idx, 1)
  else selectedRoles.value.push(role)
}
</script>

<template>
  <form class="acf" @submit.prevent="handleSubmit">
    <!-- Name -->
    <div class="acf-field">
      <label class="acf-label" for="acf-name">Name</label>
      <input
        id="acf-name"
        v-model="name"
        class="acf-input"
        :class="{ 'acf-input--error': errors.name }"
        type="text"
        placeholder="e.g. coder-agent"
        :disabled="isEdit"
        autocomplete="off"
      />
      <p v-if="errors.name" class="acf-error">{{ errors.name }}</p>
    </div>

    <!-- Roles -->
    <div class="acf-field">
      <div class="acf-label">Roles</div>
      <div class="acf-roles">
        <button
          v-for="role in availableRoles"
          :key="role"
          type="button"
          class="acf-role-chip"
          :class="{ 'acf-role-chip--selected': selectedRoles.includes(role) }"
          @click="toggleRole(role)"
        >{{ role }}</button>
      </div>
      <p v-if="errors.roles" class="acf-error">{{ errors.roles }}</p>
    </div>

    <!-- Driver -->
    <div class="acf-field">
      <div class="acf-label">Driver</div>
      <div class="acf-radio-group">
        <label class="acf-radio-label">
          <input v-model="driver" type="radio" value="claude-code-cli" />
          Claude Code
        </label>
        <label class="acf-radio-label">
          <input v-model="driver" type="radio" value="claude-mediated" />
          Claude Mediated
        </label>
        <label class="acf-radio-label">
          <input v-model="driver" type="radio" value="codex-cli" />
          Codex
        </label>
        <label class="acf-radio-label">
          <input v-model="driver" type="radio" value="gemini" />
          Gemini
        </label>
        <label class="acf-radio-label">
          <input v-model="driver" type="radio" value="gemini-cli" />
          Gemini CLI (agy)
        </label>
        <label class="acf-radio-label">
          <input v-model="driver" type="radio" value="openai-compatible" />
          OpenAI-Compatible
        </label>
      </div>
    </div>

    <!-- CLI model (Claude Code, Claude Mediated, Codex, Gemini CLI) -->
    <div
      v-if="driver === 'claude-code-cli' || driver === 'claude-mediated' || driver === 'codex-cli' || driver === 'gemini-cli'"
      class="acf-field"
    >
      <label class="acf-label" for="acf-model-cc">Model</label>
      <input
        id="acf-model-cc"
        v-model="model"
        class="acf-input"
        :class="{ 'acf-input--error': errors.model }"
        type="text"
        :placeholder="
          driver === 'codex-cli' ? 'optional, e.g. gpt-5-codex'
          : driver === 'gemini-cli' ? 'optional — agy uses its configured default'
          : 'e.g. sonnet, opus, haiku'
        "
        autocomplete="off"
      />
      <p v-if="errors.model" class="acf-error">{{ errors.model }}</p>
    </div>

    <!-- Gemini model (REST API) -->
    <div v-if="driver === 'gemini'" class="acf-field">
      <label class="acf-label" for="acf-model-gemini">Model</label>
      <input
        id="acf-model-gemini"
        v-model="model"
        class="acf-input"
        :class="{ 'acf-input--error': errors.model }"
        type="text"
        placeholder="e.g. gemini-2.5-flash, gemini-1.5-pro"
        autocomplete="off"
      />
      <p v-if="errors.model" class="acf-error">{{ errors.model }}</p>
      <p class="acf-hint">Requires <code>GEMINI_API_KEY</code> to be set in the server environment.</p>
    </div>

    <!-- OpenAI-compatible provider + model + max iterations -->
    <template v-if="driver === 'openai-compatible'">
      <div class="acf-field">
        <label class="acf-label" for="acf-provider">Provider</label>
        <div class="acf-select-row">
          <select
            id="acf-provider"
            v-model="provider"
            class="acf-select"
            :class="{ 'acf-input--error': errors.provider }"
          >
            <option value="">— select provider —</option>
            <option v-for="p in providersStore.providers" :key="p.name" :value="p.name">
              {{ p.name }} ({{ p.base_url }})
            </option>
          </select>
          <span
            v-if="provider"
            class="health-dot"
            :class="`health-dot--${providerHealth?.ok === true ? 'ok' : providerHealth?.ok === false ? 'error' : 'unknown'}`"
            :title="providerHealth?.ok ? `Connected (${providerHealth.latency_ms ?? 0} ms)` : (providerHealth?.error ?? 'Unknown')"
          />
        </div>
        <p v-if="errors.provider" class="acf-error">{{ errors.provider }}</p>
        <div v-if="providersStore.providers.length === 0" class="acf-hint">
          No providers registered. Add one in the <em>Providers</em> settings page.
        </div>
      </div>

      <div class="acf-field">
        <label class="acf-label" for="acf-provider-model">Model</label>
        <div class="acf-select-row">
          <select
            v-if="providerModels.length"
            id="acf-provider-model"
            v-model="model"
            class="acf-select"
            :class="{ 'acf-input--error': errors.model }"
          >
            <option value="">— select model —</option>
            <option v-for="m in providerModels" :key="m.id" :value="m.id">
              {{ m.name && m.name !== m.id ? `${m.name} (${m.id})` : m.id }}
            </option>
          </select>
          <input
            v-else
            id="acf-provider-model"
            v-model="model"
            class="acf-input"
            :class="{ 'acf-input--error': errors.model }"
            type="text"
            placeholder="e.g. gemma-4-26B-A4B-it-UD-Q8_K_XL or qwen3-coder:30b"
            autocomplete="off"
          />
          <button
            type="button"
            class="btn-refresh"
            :disabled="!provider || fetchingProviderModels"
            title="Refresh models from provider"
            @click="loadProviderModels"
          >{{ fetchingProviderModels ? '…' : '↻' }}</button>
        </div>
        <p v-if="errors.model" class="acf-error">{{ errors.model }}</p>
      </div>

      <div class="acf-field">
        <label class="acf-label" for="acf-max-iterations">
          Max Tool Iterations
          <span class="acf-optional">(default: 25)</span>
        </label>
        <input
          id="acf-max-iterations"
          v-model.number="maxToolIterations"
          class="acf-input acf-input--short"
          type="number"
          min="1"
          placeholder="25"
        />
      </div>
    </template>

    <!-- Failover configuration -->
    <div class="acf-section">
      <div class="acf-label">Failover Configuration</div>
      <p class="acf-hint">
        When the primary provider encounters HTTP 529 or rate limits, the runner
        can automatically failover to this provider.
      </p>
      <div class="acf-field">
        <label class="acf-label" for="acf-fallback-provider">Fallback Provider <span class="acf-optional">(optional)</span></label>
        <select
          id="acf-fallback-provider"
          v-model="fallbackProvider"
          class="acf-select"
          :class="{ 'acf-input--error': errors.fallbackProvider }"
        >
          <option value="">— none —</option>
          <option v-for="p in providersStore.providers" :key="p.name" :value="p.name">{{ p.name }}</option>
        </select>
        <p v-if="errors.fallbackProvider" class="acf-error">{{ errors.fallbackProvider }}</p>
      </div>
      <div v-if="fallbackProvider" class="acf-field">
        <label class="acf-label" for="acf-fallback-model">Fallback Model</label>
        <input
          id="acf-fallback-model"
          v-model="fallbackModel"
          class="acf-input"
          :class="{ 'acf-input--error': errors.fallbackModel }"
          type="text"
          placeholder="e.g. gemini-2.5-flash"
          autocomplete="off"
        />
        <p v-if="errors.fallbackModel" class="acf-error">{{ errors.fallbackModel }}</p>
      </div>
    </div>

    <!-- Allowed write paths -->
    <div class="acf-field">
      <label class="acf-label" for="acf-paths">
        Allowed Write Paths
        <span class="acf-optional">(one per line)</span>
      </label>
      <textarea
        id="acf-paths"
        v-model="allowedWritePathsRaw"
        class="acf-textarea"
        rows="3"
        placeholder="web/src&#10;lifecycle/frontend-plans"
      />
    </div>

    <!-- Timeout -->
    <div class="acf-field">
      <label class="acf-label" for="acf-timeout">Timeout (minutes, 0 = unlimited)</label>
      <input
        id="acf-timeout"
        v-model.number="timeoutMinutes"
        class="acf-input acf-input--short"
        type="number"
        min="0"
      />
    </div>

    <!-- Git identity -->
    <div class="acf-row">
      <div class="acf-field">
        <label class="acf-label" for="acf-git-name">Git Name</label>
        <input
          id="acf-git-name"
          v-model="gitIdentityName"
          class="acf-input"
          type="text"
          placeholder="Agent Name"
          autocomplete="off"
        />
      </div>
      <div class="acf-field">
        <label class="acf-label" for="acf-git-email">Git Email</label>
        <input
          id="acf-git-email"
          v-model="gitIdentityEmail"
          class="acf-input"
          type="email"
          placeholder="agent@example.local"
          autocomplete="off"
        />
      </div>
    </div>

    <!-- Prompt templates (one textarea per role) -->
    <div class="acf-field">
      <div class="acf-label">
        Prompt Templates
        <span class="acf-optional">(per role; clearing a body removes that template)</span>
      </div>

      <div v-for="role in promptTemplateRoles" :key="role" class="acf-field acf-prompt-entry">
        <div class="acf-prompt-entry-header">
          <span class="acf-prompt-role-name">{{ role }}</span>
          <button
            type="button"
            class="acf-prompt-remove"
            @click="removeTemplateRole(role)"
          >Remove</button>
        </div>
        <textarea
          v-model="promptTemplates[role]"
          class="acf-textarea acf-textarea--tall"
          rows="8"
          spellcheck="false"
        />
      </div>

      <div v-if="templateRolesAvailableToAdd.length" class="acf-select-row">
        <select v-model="newTemplateRole" class="acf-select">
          <option value="">— add template for role —</option>
          <option v-for="r in templateRolesAvailableToAdd" :key="r" :value="r">{{ r }}</option>
        </select>
        <button type="button" class="btn-refresh" :disabled="!newTemplateRole" @click="addTemplateRole">+ Add</button>
      </div>
    </div>

    <!-- Actions -->
    <div class="acf-actions">
      <button type="button" class="btn-secondary" @click="emit('cancel')">Cancel</button>
      <button type="submit" class="btn-primary">{{ isEdit ? 'Save Changes' : 'Create Agent' }}</button>
    </div>
  </form>
</template>

<style scoped>
.acf {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.acf-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.acf-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}
.acf-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}
.acf-label {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}
.acf-optional {
  font-weight: 400;
  color: var(--color-text-muted);
  font-size: 12px;
}
.acf-input {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  outline: none;
}
.acf-input:focus { border-color: var(--color-accent); }
.acf-input--error { border-color: var(--color-error); }
.acf-input--short { max-width: 120px; }
.acf-input:disabled { opacity: 0.6; cursor: not-allowed; }
.acf-select {
  flex: 1;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: var(--text-sm);
  outline: none;
}
.acf-select:focus { border-color: var(--color-accent); }
.acf-select-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.acf-textarea {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: 13px;
  font-family: monospace;
  outline: none;
  resize: vertical;
}
.acf-textarea:focus { border-color: var(--color-accent); }
.acf-textarea--tall { min-height: 160px; }
.acf-prompt-entry {
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}
.acf-prompt-entry-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.acf-prompt-role-name {
  font-size: var(--text-sm);
  font-weight: 500;
  font-family: monospace;
}
.acf-prompt-remove {
  background: transparent;
  border: none;
  color: var(--color-text-muted);
  font-size: 12px;
  cursor: pointer;
  padding: 0;
}
.acf-prompt-remove:hover { color: var(--color-error); }
.acf-error {
  font-size: 12px;
  color: var(--color-error);
  margin: 0;
}
.acf-hint {
  font-size: 12px;
  color: var(--color-text-muted);
  margin: 0;
}
.acf-radio-group {
  display: flex;
  gap: var(--space-4);
}
.acf-radio-label {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--text-sm);
  cursor: pointer;
}
.acf-roles {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}
.acf-role-chip {
  padding: 2px 10px;
  border: 1px solid var(--color-border);
  border-radius: 99px;
  font-size: 12px;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: border-color 0.1s, background 0.1s, color 0.1s;
}
.acf-role-chip:hover { border-color: var(--color-accent); color: var(--color-text); }
.acf-role-chip--selected {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
}
.health-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
.health-dot--ok    { background: #22c55e; }
.health-dot--error { background: #ef4444; }
.health-dot--unknown { background: var(--color-border); }
.btn-refresh {
  padding: 4px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-muted);
  font-size: 14px;
  cursor: pointer;
}
.btn-refresh:hover:not(:disabled) { border-color: var(--color-accent); color: var(--color-accent); }
.btn-refresh:disabled { opacity: 0.4; cursor: not-allowed; }
.acf-actions {
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
