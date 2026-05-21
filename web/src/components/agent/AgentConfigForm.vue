<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useOllamaInstancesStore } from '@/stores/ollamaInstances'
import type { AgentSummary, OllamaInstance } from '@/types/api'

// The shape the form emits — mirrors the YAML agent config fields.
export interface AgentFormData {
  name: string
  roles: string[]
  driver: 'claude-code-cli' | 'ollama' | 'gemini' | 'gemini-cli'
  model: string
  ollama_instance: string
  ollama_endpoint: 'chat' | 'generate'
  allowed_write_paths: string[]
  timeout_minutes: number
  git_identity_name: string
  git_identity_email: string
  prompt_templates: Record<string, string>
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

const ollamaStore = useOllamaInstancesStore()

const isEdit = !!props.initial

// ── Form state ─────────────────────────────────────────────────────────────
const name = ref(props.initial?.name ?? '')
const selectedRoles = ref<string[]>(props.initial?.roles ?? [])
const driver = ref<'claude-code-cli' | 'ollama' | 'gemini' | 'gemini-cli'>(
  (props.initial?.driver ?? 'claude-code-cli') as 'claude-code-cli' | 'ollama' | 'gemini' | 'gemini-cli',
)
const model = ref(props.initial?.model ?? '')
const ollamaInstance = ref(props.initial?.ollama_instance ?? '')
const ollamaEndpoint = ref<'chat' | 'generate'>(
  (props.initial?.ollama_endpoint ?? 'chat') as 'chat' | 'generate',
)
const allowedWritePathsRaw = ref((props.initial?.allowed_write_paths ?? []).join('\n'))
const timeoutMinutes = ref(0)
const gitIdentityName = ref('')
const gitIdentityEmail = ref('')
const promptTemplatesRaw = ref('')
const errors = ref<Record<string, string>>({})

// ── Ollama model list ───────────────────────────────────────────────────────
const instanceModels = computed(() => {
  if (!ollamaInstance.value) return []
  return ollamaStore.models.get(ollamaInstance.value) ?? []
})

const instanceHealth = computed(() => {
  if (!ollamaInstance.value) return null
  return ollamaStore.health.get(ollamaInstance.value) ?? null
})

const fetchingModels = ref(false)

async function loadModels() {
  if (!ollamaInstance.value) return
  fetchingModels.value = true
  try {
    await ollamaStore.fetchModels(ollamaInstance.value)
  } finally {
    fetchingModels.value = false
  }
}

watch(ollamaInstance, (val) => {
  if (val) loadModels()
  model.value = ''
})

onMounted(async () => {
  if (!ollamaStore.instances.length) {
    await ollamaStore.fetchInstances()
  }
  await ollamaStore.checkAllHealth()
  if (ollamaInstance.value) {
    await loadModels()
  }
})

// ── Validation ─────────────────────────────────────────────────────────────
function validate(): boolean {
  const e: Record<string, string> = {}
  if (!name.value.trim()) e.name = 'Name is required.'
  else if (!isEdit && props.existingNames?.includes(name.value.trim()))
    e.name = 'An agent with this name already exists.'
  if (!selectedRoles.value.length) e.roles = 'At least one role is required.'
  if (driver.value === 'ollama') {
    if (!ollamaInstance.value) e.ollama_instance = 'Select an Ollama instance.'
    if (!model.value.trim()) e.model = 'Model is required for Ollama driver.'
  } else if (driver.value !== 'gemini-cli') {
    if (!model.value.trim()) e.model = 'Model is required.'
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
    ollama_instance: ollamaInstance.value,
    ollama_endpoint: ollamaEndpoint.value,
    allowed_write_paths: allowedWritePathsRaw.value
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean),
    timeout_minutes: timeoutMinutes.value,
    git_identity_name: gitIdentityName.value.trim(),
    git_identity_email: gitIdentityEmail.value.trim(),
    prompt_templates: parsePromptTemplates(),
  })
}

function parsePromptTemplates(): Record<string, string> {
  // Simple format: lines starting with "role:" introduce a template block.
  const result: Record<string, string> = {}
  if (!promptTemplatesRaw.value.trim()) return result
  try {
    // Expect: role-name: | followed by indented text
    const lines = promptTemplatesRaw.value.split('\n')
    let currentRole = ''
    const currentLines: string[] = []
    for (const line of lines) {
      const match = line.match(/^([a-z0-9-]+):\s*\|?\s*$/)
      if (match) {
        if (currentRole) result[currentRole] = currentLines.join('\n').trimEnd()
        currentRole = match[1]
        currentLines.length = 0
      } else if (currentRole) {
        currentLines.push(line.replace(/^  /, ''))
      }
    }
    if (currentRole) result[currentRole] = currentLines.join('\n').trimEnd()
  } catch {
    // ignore parse errors
  }
  return result
}

function toggleRole(role: string) {
  const idx = selectedRoles.value.indexOf(role)
  if (idx >= 0) selectedRoles.value.splice(idx, 1)
  else selectedRoles.value.push(role)
}

function healthDot(inst: OllamaInstance): 'ok' | 'error' | 'unknown' {
  const h = ollamaStore.health.get(inst.name)
  if (!h) return 'unknown'
  return h.ok ? 'ok' : 'error'
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
        placeholder="e.g. my-ollama-agent"
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
          <input v-model="driver" type="radio" value="ollama" />
          Ollama
        </label>
        <label class="acf-radio-label">
          <input v-model="driver" type="radio" value="gemini" />
          Gemini
        </label>
        <label class="acf-radio-label">
          <input v-model="driver" type="radio" value="gemini-cli" />
          Gemini CLI (agy)
        </label>
      </div>
    </div>

    <!-- Claude Code model -->
    <div v-if="driver === 'claude-code-cli'" class="acf-field">
      <label class="acf-label" for="acf-model-cc">Model</label>
      <input
        id="acf-model-cc"
        v-model="model"
        class="acf-input"
        :class="{ 'acf-input--error': errors.model }"
        type="text"
        placeholder="e.g. sonnet, opus, haiku"
        autocomplete="off"
      />
      <p v-if="errors.model" class="acf-error">{{ errors.model }}</p>
    </div>

    <!-- Gemini model -->
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
    </div>

    <!-- Ollama instance + model -->
    <template v-if="driver === 'ollama'">
      <div class="acf-field">
        <label class="acf-label" for="acf-ollama-instance">Ollama Instance</label>
        <div class="acf-select-row">
          <select
            id="acf-ollama-instance"
            v-model="ollamaInstance"
            class="acf-select"
            :class="{ 'acf-input--error': errors.ollama_instance }"
          >
            <option value="">— select instance —</option>
            <option v-for="inst in ollamaStore.instances" :key="inst.name" :value="inst.name">
              {{ inst.name }}
              ({{ inst.base_url }})
            </option>
          </select>
          <span
            v-if="ollamaInstance"
            class="health-dot"
            :class="`health-dot--${instanceHealth?.ok === true ? 'ok' : instanceHealth?.ok === false ? 'error' : 'unknown'}`"
            :title="instanceHealth?.ok ? 'Connected' : (instanceHealth?.error ?? 'Unknown')"
          />
        </div>
        <p v-if="errors.ollama_instance" class="acf-error">{{ errors.ollama_instance }}</p>
        <div v-if="ollamaStore.instances.length === 0" class="acf-hint">
          No Ollama instances registered. Add one in the <em>Ollama</em> settings page.
        </div>
      </div>

      <div class="acf-field">
        <label class="acf-label" for="acf-ollama-model">Model</label>
        <div class="acf-select-row">
          <select
            v-if="instanceModels.length"
            id="acf-ollama-model"
            v-model="model"
            class="acf-select"
            :class="{ 'acf-input--error': errors.model }"
          >
            <option value="">— select model —</option>
            <option v-for="m in instanceModels" :key="m.name" :value="m.name">
              {{ m.name }}
            </option>
          </select>
          <input
            v-else
            id="acf-ollama-model"
            v-model="model"
            class="acf-input"
            :class="{ 'acf-input--error': errors.model }"
            type="text"
            placeholder="e.g. llama3:8b"
            autocomplete="off"
          />
          <button
            type="button"
            class="btn-refresh"
            :disabled="!ollamaInstance || fetchingModels"
            @click="loadModels"
          >{{ fetchingModels ? '…' : '↻' }}</button>
        </div>
        <p v-if="errors.model" class="acf-error">{{ errors.model }}</p>
      </div>

      <div class="acf-field">
        <div class="acf-label">Endpoint</div>
        <div class="acf-radio-group">
          <label class="acf-radio-label">
            <input v-model="ollamaEndpoint" type="radio" value="chat" />
            /api/chat (default)
          </label>
          <label class="acf-radio-label">
            <input v-model="ollamaEndpoint" type="radio" value="generate" />
            /api/generate
          </label>
        </div>
      </div>
    </template>

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

    <!-- Prompt templates (simple text editor) -->
    <div class="acf-field">
      <label class="acf-label" for="acf-prompts">
        Prompt Templates
        <span class="acf-optional">(role-name: | followed by indented text)</span>
      </label>
      <textarea
        id="acf-prompts"
        v-model="promptTemplatesRaw"
        class="acf-textarea acf-textarea--tall"
        rows="8"
        placeholder="frontend-developer: |&#10;  You are a frontend developer…"
        spellcheck="false"
      />
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
