<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { ArtifactFrontmatter, ArtifactAssignee } from '@/types/api'
import { useProjectConfigStore } from '@/stores/projectConfig'
import { useReleasesStore } from '@/stores/releases'
import AssigneeEditor from './AssigneeEditor.vue'
import ReleaseFormModal from '@/components/releases/ReleaseFormModal.vue'

const props = defineProps<{ modelValue: ArtifactFrontmatter; project: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: ArtifactFrontmatter] }>()

const STATUS_VOCAB = [
  'abandoned',
  'approved',
  'blocked',
  'clarifying',
  'done',
  'draft',
  'in-development',
  'in-qa',
  'planning',
  'raw',
  'rejected',
] as const

const projectConfigStore = useProjectConfigStore()
const releasesStore = useReleasesStore()

const showCreateReleaseModal = ref(false)

onMounted(() => {
  projectConfigStore.fetchRoles(props.project)
  if (releasesStore.releases.length === 0) {
    releasesStore.fetch(props.project)
  }
})

function onReleaseSelectChange(event: Event) {
  const val = (event.target as HTMLSelectElement).value
  if (val === '__create__') {
    showCreateReleaseModal.value = true
    // Reset select to current value
    ;(event.target as HTMLSelectElement).value = props.modelValue.release ?? ''
  } else {
    update('release', val || undefined)
  }
}

function onReleaseCreated(release: import('@/types/release').Release) {
  showCreateReleaseModal.value = false
  update('release', release.name)
}

function update<K extends keyof ArtifactFrontmatter>(field: K, value: ArtifactFrontmatter[K]) {
  emit('update:modelValue', { ...props.modelValue, [field]: value })
}

function parseList(s: string): string[] {
  return s.split(',').map((v) => v.trim()).filter(Boolean)
}

function formatList(arr: string[] | undefined): string {
  return (arr ?? []).join(', ')
}

const statusIsUnknown = computed(() => !STATUS_VOCAB.includes(props.modelValue.status as typeof STATUS_VOCAB[number]))

function updateAssignees(assignees: ArtifactAssignee[]) {
  update('assignees', assignees.length > 0 ? assignees : undefined)
}

// Validation: returns index of first invalid assignee row, or -1 if all valid.
function firstInvalidAssigneeIndex(): number {
  const assignees = props.modelValue.assignees ?? []
  return assignees.findIndex((a) => !a.role.trim() || !a.who.trim())
}

// Expose validation so the parent editor view can call it before saving.
defineExpose({ firstInvalidAssigneeIndex })

const assigneeError = computed(() => {
  const idx = firstInvalidAssigneeIndex()
  if (idx === -1) return null
  return `Assignee row ${idx + 1}: both role and who are required.`
})
</script>

<template>
  <aside class="fm-editor">
    <h3 class="fm-title">Frontmatter</h3>
    <div class="fm-fields">

      <label class="fm-field">
        <span class="fm-label">Title</span>
        <input
          class="fm-input"
          type="text"
          :value="modelValue.title"
          @input="update('title', ($event.target as HTMLInputElement).value)"
        />
      </label>

      <label class="fm-field">
        <span class="fm-label">Status</span>
        <select
          class="fm-input fm-select"
          :value="modelValue.status"
          @change="update('status', ($event.target as HTMLSelectElement).value)"
        >
          <option
            v-if="statusIsUnknown"
            :value="modelValue.status"
            disabled
          >{{ modelValue.status }}</option>
          <option v-for="s in STATUS_VOCAB" :key="s" :value="s">{{ s }}</option>
        </select>
      </label>

      <label class="fm-field">
        <span class="fm-label">Priority</span>
        <select
          class="fm-input fm-select"
          :value="modelValue.priority ?? ''"
          @change="update('priority', ($event.target as HTMLSelectElement).value || undefined)"
        >
          <option value="">— none —</option>
          <option value="normal">normal</option>
          <option value="low">low</option>
          <option value="medium">medium</option>
          <option value="high">high</option>
        </select>
      </label>

      <div class="fm-field fm-readonly">
        <span class="fm-label">Type</span>
        <span class="fm-value">{{ modelValue.type }}</span>
      </div>

      <div class="fm-field fm-readonly">
        <span class="fm-label">Lineage</span>
        <span class="fm-value mono">{{ modelValue.lineage }}</span>
      </div>

      <label class="fm-field">
        <span class="fm-label">Labels</span>
        <input
          class="fm-input"
          type="text"
          :value="formatList(modelValue.labels)"
          placeholder="comma-separated"
          @change="update('labels', parseList(($event.target as HTMLInputElement).value))"
        />
      </label>

      <div class="fm-field">
        <span class="fm-label">Release</span>
        <select
          class="fm-input fm-select"
          :value="modelValue.release ?? ''"
          @change="onReleaseSelectChange"
        >
          <option value="">Unassigned</option>
          <option
            v-for="r in releasesStore.releases"
            :key="r.id"
            :value="r.name"
          >{{ r.name }}</option>
          <option value="__create__">+ Create Release</option>
        </select>
      </div>

      <ReleaseFormModal
        v-if="showCreateReleaseModal"
        :project="props.project"
        @saved="onReleaseCreated"
        @close="showCreateReleaseModal = false"
      />

      <label class="fm-field">
        <span class="fm-label">Sprint</span>
        <input
          class="fm-input"
          type="text"
          :value="modelValue.sprint ?? ''"
          @input="update('sprint', ($event.target as HTMLInputElement).value || undefined)"
        />
      </label>

      <label class="fm-field">
        <span class="fm-label">Depends on</span>
        <input
          class="fm-input"
          type="text"
          :value="formatList(modelValue.depends_on)"
          placeholder="comma-separated paths"
          @change="update('depends_on', parseList(($event.target as HTMLInputElement).value))"
        />
      </label>

      <label class="fm-field">
        <span class="fm-label">Blocks</span>
        <input
          class="fm-input"
          type="text"
          :value="formatList(modelValue.blocks)"
          placeholder="comma-separated paths"
          @change="update('blocks', parseList(($event.target as HTMLInputElement).value))"
        />
      </label>

      <div class="fm-field">
        <span class="fm-label">Assignees</span>
        <AssigneeEditor
          :model-value="modelValue.assignees ?? []"
          :roles="projectConfigStore.roles"
          :who-options="projectConfigStore.availableWhoOptions"
          @update:model-value="updateAssignees"
        />
        <span v-if="assigneeError" class="fm-error">{{ assigneeError }}</span>
      </div>

    </div>
  </aside>
</template>

<style scoped>
.fm-editor {
  width: 220px;
  flex-shrink: 0;
  background: var(--color-surface);
  border-left: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  padding: var(--space-4);
}
.fm-title {
  font-size: var(--text-sm);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  margin: 0 0 var(--space-4);
}
.fm-fields {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.fm-field {
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.fm-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
}
.fm-input {
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
.fm-input:focus {
  outline: none;
  border-color: var(--color-accent);
}
.fm-select {
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M0 0l5 6 5-6z' fill='%23888'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right var(--space-2) center;
  padding-right: calc(var(--space-2) * 2 + 10px);
  cursor: pointer;
}
.fm-readonly {
  opacity: 0.6;
}
.fm-value {
  font-size: var(--text-sm);
  color: var(--color-text);
}
.mono { font-family: monospace; font-size: 12px; }
.fm-error {
  font-size: 11px;
  color: #dc2626;
  margin-top: 2px;
}
</style>
