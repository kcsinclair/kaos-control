<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import * as releasesApi from '@/api/releases'
import type { ReleaseDetail } from '@/types/release'
import type { ArtifactRow } from '@/types/api'

const props = defineProps<{
  releaseId: number
  project: string
}>()

const emit = defineEmits<{
  close: []
  edit: []
  delete: []
}>()

const router = useRouter()

const detail = ref<ReleaseDetail | null>(null)
const artifacts = ref<ArtifactRow[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

function formatDate(d: string | null): string {
  if (!d) return '—'
  return new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

function statusLabel(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1)
}

function artifactTypeBadgeClass(type: string): string {
  const map: Record<string, string> = {
    idea: 'badge--idea',
    defect: 'badge--defect',
  }
  return map[type] ?? 'badge--default'
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const [d, arts] = await Promise.all([
      releasesApi.getRelease(props.project, props.releaseId),
      releasesApi.listReleaseArtifacts(props.project, props.releaseId),
    ])
    detail.value = d
    artifacts.value = arts ?? []
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load release.'
  } finally {
    loading.value = false
  }
}

function openArtifact(artifact: ArtifactRow) {
  router.push(`/p/${encodeURIComponent(props.project)}/artifacts/${artifact.path}`)
  emit('close')
}

function openFile() {
  if (detail.value?.file_path) {
    router.push(`/p/${encodeURIComponent(props.project)}/artifacts/${detail.value.file_path}`)
    emit('close')
  }
}

const filteredArtifacts = computed(() =>
  artifacts.value.filter(a => a.type === 'idea' || a.type === 'defect')
)

onMounted(load)
</script>

<template>
  <div class="modal-overlay">
    <div class="modal-panel" role="dialog" aria-modal="true" aria-label="Release detail">
      <div class="modal-header">
        <h3 class="modal-title">{{ detail?.name ?? 'Release' }}</h3>
        <button class="btn-icon" aria-label="Close" @click="emit('close')">✕</button>
      </div>

      <div class="modal-body">
        <div v-if="loading" class="state-msg">Loading…</div>
        <div v-else-if="error" class="state-msg state-msg--error">{{ error }}</div>

        <template v-else-if="detail">
          <div class="detail-meta">
            <div class="meta-item">
              <span class="meta-label">Status</span>
              <span class="status-badge" :class="`status-badge--${detail.status}`">
                {{ statusLabel(detail.status) }}
              </span>
            </div>
            <div class="meta-item">
              <span class="meta-label">Start</span>
              <span class="meta-value">{{ formatDate(detail.start_date) }}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">End</span>
              <span class="meta-value">{{ formatDate(detail.end_date) }}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">Ideas</span>
              <span class="meta-value">{{ detail.idea_count }}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">Defects</span>
              <span class="meta-value">{{ detail.defect_count }}</span>
            </div>
            <div v-if="detail.file_path" class="meta-item meta-item--full">
              <span class="meta-label">File</span>
              <button class="file-path-link" @click="openFile">{{ detail.file_path }}</button>
            </div>
          </div>

          <div class="artifacts-section">
            <h4 class="artifacts-heading">Assigned Artifacts ({{ filteredArtifacts.length }})</h4>
            <div v-if="filteredArtifacts.length === 0" class="state-msg">No artifacts assigned.</div>
            <div v-else class="artifact-list">
              <button
                v-for="artifact in filteredArtifacts"
                :key="artifact.path"
                class="artifact-card"
                @click="openArtifact(artifact)"
              >
                <div class="artifact-row">
                  <span class="type-badge" :class="artifactTypeBadgeClass(artifact.type)">{{ artifact.type }}</span>
                  <span class="artifact-title">{{ artifact.title }}</span>
                  <span class="status-chip" :data-status="artifact.status">{{ artifact.status }}</span>
                </div>
                <div class="artifact-lineage">{{ artifact.lineage }}</div>
              </button>
            </div>
          </div>
        </template>
      </div>

      <div class="modal-footer">
        <button class="btn-secondary" @click="emit('edit')">Edit</button>
        <button class="btn-danger-outline" @click="emit('delete')">Delete</button>
        <button class="btn-ghost ml-auto" @click="emit('close')">Close</button>
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
  width: 540px;
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
  gap: var(--space-4);
}
.state-msg {
  text-align: center;
  padding: var(--space-6) 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.state-msg--error { color: #dc2626; }
.detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-4);
}
.meta-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.meta-label {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
}
.meta-value {
  font-size: var(--text-sm);
  color: var(--color-text);
}
.status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
}
.status-badge--planned     { background: #e2e8f0; color: #334155; }
.status-badge--active      { background: #dbeafe; color: #1e40af; }
.status-badge--shipped     { background: #d1fae5; color: #065f46; }
.status-badge--unscheduled { background: #f1f5f9; color: #64748b; }
@media (prefers-color-scheme: dark) {
  .status-badge--planned     { background: #334155; color: #e2e8f0; }
  .status-badge--active      { background: #1e3a5f; color: #93c5fd; }
  .status-badge--shipped     { background: #064e3b; color: #6ee7b7; }
  .status-badge--unscheduled { background: #1e293b; color: #94a3b8; }
}
.meta-item--full { width: 100%; }
.file-path-link {
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
.file-path-link:hover {
  background: var(--color-bg);
  text-decoration-style: solid;
}
.artifacts-section { display: flex; flex-direction: column; gap: var(--space-2); }
.artifacts-heading {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}
.artifact-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.artifact-card {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: none;
  text-align: left;
  cursor: pointer;
  font-family: inherit;
  transition: background 0.12s, border-color 0.12s;
}
.artifact-card:hover {
  background: var(--color-surface);
  border-color: var(--color-accent);
}
.artifact-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}
.artifact-title {
  flex: 1;
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
  text-align: left;
}
.artifact-lineage {
  font-size: 11px;
  font-family: monospace;
  color: var(--color-text-muted);
}
.type-badge {
  display: inline-block;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  flex-shrink: 0;
}
.badge--idea    { background: #ede9fe; color: #5b21b6; }
.badge--defect  { background: #fee2e2; color: #991b1b; }
.badge--default { background: #f1f5f9; color: #334155; }
@media (prefers-color-scheme: dark) {
  .badge--idea    { background: #3b2f6e; color: #c4b5fd; }
  .badge--defect  { background: #7f1d1d; color: #fca5a5; }
  .badge--default { background: #1f2937; color: #cbd5e1; }
}

.status-chip {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 99px;
  font-size: 10px;
  font-weight: 500;
  flex-shrink: 0;
}
/* Per-status palette — keep in sync with web/src/components/artifact/StatusDropdown.vue */
.status-chip[data-status="raw"]            { background: #f1f5f9; color: #475569; }
.status-chip[data-status="draft"]          { background: #f3f4f6; color: #374151; }
.status-chip[data-status="clarifying"]     { background: #ede9fe; color: #5b21b6; }
.status-chip[data-status="planning"]       { background: #fef3c7; color: #92400e; }
.status-chip[data-status="in-development"] { background: #dbeafe; color: #1e40af; }
.status-chip[data-status="in-qa"]          { background: #ede9fe; color: #6d28d9; }
.status-chip[data-status="approved"]       { background: #d1fae5; color: #065f46; }
.status-chip[data-status="done"]           { background: #bbf7d0; color: #14532d; }
.status-chip[data-status="blocked"]        { background: #fee2e2; color: #991b1b; }
.status-chip[data-status="rejected"]       { background: #fef2f2; color: #b91c1c; }
.status-chip[data-status="abandoned"]      { background: #f3f4f6; color: #6b7280; }
.status-chip[data-status="in-progress"]    { background: #fef3c7; color: #92400e; }
@media (prefers-color-scheme: dark) {
  .status-chip[data-status="raw"]            { background: #1e293b; color: #cbd5e1; }
  .status-chip[data-status="draft"]          { background: #374151; color: #d1d5db; }
  .status-chip[data-status="clarifying"]     { background: #3b2f6e; color: #c4b5fd; }
  .status-chip[data-status="planning"]       { background: #422006; color: #fcd34d; }
  .status-chip[data-status="in-development"] { background: #1e3a5f; color: #93c5fd; }
  .status-chip[data-status="in-qa"]          { background: #2e1065; color: #c4b5fd; }
  .status-chip[data-status="approved"]       { background: #064e3b; color: #6ee7b7; }
  .status-chip[data-status="done"]           { background: #052e16; color: #4ade80; }
  .status-chip[data-status="blocked"]        { background: #7f1d1d; color: #fca5a5; }
  .status-chip[data-status="rejected"]       { background: #7f1d1d; color: #fca5a5; }
  .status-chip[data-status="abandoned"]      { background: #1f2937; color: #9ca3af; }
  .status-chip[data-status="in-progress"]    { background: #422006; color: #fcd34d; }
}
.modal-footer {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-4) var(--space-6);
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
  align-items: center;
}
.ml-auto { margin-left: auto; }
.btn-secondary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
  cursor: pointer;
}
.btn-secondary:hover { background: var(--color-bg); }
.btn-danger-outline {
  padding: var(--space-2) var(--space-4);
  background: none;
  border: 1px solid #dc2626;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  color: #dc2626;
  cursor: pointer;
}
.btn-danger-outline:hover { background: #fee2e2; }
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
</style>
