<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import * as configApi from '@/api/config'

const route = useRoute()
const ui = useUiStore()
const project = route.params.project as string

const raw = ref('')
const loading = ref(false)
const saving = ref(false)
const loadError = ref<string | null>(null)
const dirty = ref(false)

async function load() {
  loading.value = true
  loadError.value = null
  try {
    const res = await configApi.getConfig(project)
    raw.value = res.raw
    dirty.value = false
  } catch (e: unknown) {
    loadError.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await configApi.updateConfig(project, raw.value)
    dirty.value = false
    ui.success('Config saved')
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Save failed')
  } finally {
    saving.value = false
  }
}

function onInput(e: Event) {
  raw.value = (e.target as HTMLTextAreaElement).value
  dirty.value = true
}

onMounted(load)
</script>

<template>
  <div class="config-view">
    <div class="view-header">
      <div class="header-left">
        <h2 class="view-title">Project Config</h2>
        <span class="config-path">lifecycle/config.yaml</span>
      </div>
      <div class="header-actions">
        <span v-if="dirty" class="unsaved-hint">Unsaved changes</span>
        <button
          class="btn-primary"
          :disabled="saving || !dirty"
          @click="save"
          aria-label="Save config"
        >{{ saving ? 'Saving…' : 'Save' }}</button>
      </div>
    </div>

    <div v-if="loading" class="state-msg" role="status" aria-live="polite">Loading…</div>
    <div v-else-if="loadError" class="state-msg error" role="alert">{{ loadError }}</div>

    <div v-else class="editor-wrap">
      <textarea
        class="yaml-editor"
        :value="raw"
        @input="onInput"
        spellcheck="false"
        autocomplete="off"
        autocorrect="off"
        autocapitalize="off"
        aria-label="Project config YAML"
        aria-multiline="true"
      />
    </div>
  </div>
</template>

<style scoped>
.config-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.view-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
  gap: var(--space-4);
}
.header-left {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.view-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.config-path {
  font-size: var(--text-xs);
  font-family: monospace;
  color: var(--color-text-muted);
}
.header-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-shrink: 0;
}
.unsaved-hint {
  font-size: var(--text-sm);
  color: var(--color-warning);
}
.btn-primary {
  padding: var(--space-1) var(--space-4);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-primary:hover:not(:disabled) { opacity: 0.88; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.state-msg {
  padding: var(--space-8) var(--space-6);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.state-msg.error { color: var(--color-error); }
.editor-wrap {
  flex: 1;
  display: flex;
  overflow: hidden;
  padding: var(--space-4);
}
.yaml-editor {
  flex: 1;
  resize: none;
  font-family: 'SFMono-Regular', 'Consolas', 'Liberation Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
}
.yaml-editor:focus {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px var(--color-accent-subtle);
}
</style>
