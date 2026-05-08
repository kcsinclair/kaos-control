<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useArtifactsStore } from '@/stores/artifacts'
import { useUiStore } from '@/stores/ui'
import { useLock } from '@/composables/useLock'
import { useExternalChange } from '@/composables/useExternalChange'
import { useWebSocket } from '@/composables/useWebSocket'
import * as artifactsApi from '@/api/artifacts'
import * as agentsApi from '@/api/agents'
import LineageBreadcrumb from '@/components/artifact/LineageBreadcrumb.vue'
import FrontmatterPanel from '@/components/artifact/FrontmatterPanel.vue'
import FrontmatterEditor from '@/components/artifact/FrontmatterEditor.vue'
import MarkdownPreview from '@/components/artifact/MarkdownPreview.vue'
import MarkdownEditor from '@/components/artifact/MarkdownEditor.vue'
import LockBanner from '@/components/common/LockBanner.vue'
import RunAgentDialog from '@/components/agent/RunAgentDialog.vue'
import { useGraphStore } from '@/stores/graph'
import type { ArtifactDetail, ArtifactFrontmatter, WsEvent } from '@/types/api'

const route = useRoute()
const router = useRouter()
const store = useArtifactsStore()
const ui = useUiStore()
const graphStore = useGraphStore()

const project = computed(() => route.params.project as string)
const artifactPath = computed(() => {
  const m = route.params.pathMatch
  return Array.isArray(m) ? m.join('/') : (m as string)
})

// ── artifact state ──────────────────────────────────────────────────────────
const artifact = ref<ArtifactDetail | null>(null)
const loading = ref(false)
const loadError = ref<string | null>(null)

async function load() {
  if (!artifactPath.value) return
  loading.value = true
  loadError.value = null
  try {
    store.invalidate(artifactPath.value)
    artifact.value = await store.fetchOne(project.value, artifactPath.value)
  } catch (e: unknown) {
    loadError.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

// ── toolbar dialogs ──────────────────────────────────────────────────────────
const showRunAgent = ref(false)

// ── run test (single execution) ──────────────────────────────────────────────
const runTestRunning = ref(false)
const isTestArtifact = computed(
  () => artifact.value?.type === 'test' && artifact.value?.status === 'approved',
)

async function runTest() {
  if (!artifact.value || runTestRunning.value) return
  runTestRunning.value = true
  try {
    await agentsApi.startRun(project.value, 'qa', artifactPath.value)
  } catch (e: unknown) {
    ui.error(e instanceof Error ? e.message : 'Failed to start test run')
    runTestRunning.value = false
  }
}

// Reset running state when the agent finishes/fails for this artifact
useWebSocket(project.value, 'agent.finished', (e) => {
  const tp = e.payload?.target_path as string | undefined
  if (tp === artifactPath.value) runTestRunning.value = false
})
useWebSocket(project.value, 'agent.failed', (e) => {
  const tp = e.payload?.target_path as string | undefined
  if (tp === artifactPath.value) runTestRunning.value = false
})

// ── stale in-qa warning ──────────────────────────────────────────────────────
const isStale = ref(false)

useWebSocket(project.value, 'test.stale', (e) => {
  const path = e.payload?.path as string | undefined
  if (path === artifactPath.value) isStale.value = true
})

// ── edit mode state ─────────────────────────────────────────────────────────
const editing = ref(false)
const saving = ref(false)
const editBody = ref('')
const editFrontmatter = ref<ArtifactFrontmatter | null>(null)

// ── lock ────────────────────────────────────────────────────────────────────
const hasOpenQuestions = computed(() =>
  !!artifact.value && /^## Open Questions/m.test(artifact.value.body),
)

function scrollToOpenQuestions() {
  const preview = document.querySelector('.editor-content .md-preview')
  if (!preview) return
  for (const h of preview.querySelectorAll('h2')) {
    if (h.textContent?.trim() === 'Open Questions') {
      h.scrollIntoView({ behavior: 'smooth' })
      return
    }
  }
}

const { acquired: lockAcquired, conflictLock, acquire: acquireLock, release: releaseLock } = useLock(
  project.value,
  computed(() => artifact.value?.lineage ?? '').value,
)

// ── external change detection ────────────────────────────────────────────────
// Track when auto-refresh last completed so the artifact.indexed listener
// can skip a redundant fetch for the same change event.
const AUTO_REFRESH_GRACE_MS = 2_000
let lastAutoRefreshMs = 0

async function autoRefresh() {
  store.invalidate(artifactPath.value)
  artifact.value = await store.fetchOne(project.value, artifactPath.value)
  lastAutoRefreshMs = Date.now()
  ui.info('File updated on disk')
}

const { hasExternalChange, markSaved, acknowledge } = useExternalChange(
  project.value,
  artifactPath.value,
  { isDirty: () => editing.value, onAutoRefresh: autoRefresh },
)

// ── enter / exit edit mode ──────────────────────────────────────────────────
async function enterEdit() {
  if (!artifact.value) return
  if (conflictLock.value) return // locked by other
  const ok = await acquireLock()
  if (!ok && conflictLock.value) return // failed due to conflict
  editBody.value = artifact.value.body
  editFrontmatter.value = { ...artifact.value.frontmatter }
  editing.value = true
}

async function cancelEdit() {
  editing.value = false
  await releaseLock()
  // Reset working copies
  editBody.value = ''
  editFrontmatter.value = null
}

// ── save ─────────────────────────────────────────────────────────────────────
async function save() {
  if (!artifact.value || !editFrontmatter.value) return
  const invalidAssignee = (editFrontmatter.value.assignees ?? []).findIndex(
    (a) => !a.role.trim() || !a.who.trim(),
  )
  if (invalidAssignee !== -1) {
    ui.error(`Assignee row ${invalidAssignee + 1}: both role and who are required.`)
    return
  }
  const submittedStatus = editFrontmatter.value.status
  saving.value = true
  try {
    markSaved()
    await artifactsApi.updateArtifact(project.value, artifactPath.value, {
      frontmatter: editFrontmatter.value,
      body: editBody.value,
      expected_sha: artifact.value.file_sha,
    })
    // Reload fresh (including new sha)
    store.invalidate(artifactPath.value)
    artifact.value = await store.fetchOne(project.value, artifactPath.value)
    editing.value = false
    await releaseLock()
    if (artifact.value?.status === 'blocked' && submittedStatus !== 'blocked') {
      ui.info('Status changed to blocked — open questions require product-owner review.')
    } else {
      ui.success('Saved')
    }
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : 'Save failed'
    if (msg.includes('modified since last read') || msg.includes('conflict')) {
      ui.error('Conflict: the artifact was modified externally. Reload to get the latest version.')
    } else {
      ui.error(msg)
    }
  } finally {
    saving.value = false
  }
}

// ── external change: reload or keep editing ──────────────────────────────────
async function reloadFromDisk() {
  acknowledge()
  store.invalidate(artifactPath.value)
  artifact.value = await store.fetchOne(project.value, artifactPath.value)
  if (editing.value && artifact.value) {
    editBody.value = artifact.value.body
    editFrontmatter.value = { ...artifact.value.frontmatter }
  }
}

// ── WS: re-index by agent or external tool ───────────────────────────────────
// Reactive chain: load() fetches the full artifact including body; hasOpenQuestions
// is a computed that reads artifact.value.body, so it recomputes automatically once
// load() resolves — the blocked banner appears/disappears atomically with the status
// badge without any manual refresh.
useWebSocket(project.value, 'artifact.indexed', async (e: WsEvent) => {
  if (e.payload?.path !== artifactPath.value || editing.value) return
  // Skip if auto-refresh already handled this change (file.changed + re-fetch)
  if (Date.now() - lastAutoRefreshMs < AUTO_REFRESH_GRACE_MS) return
  const prevStatus = artifact.value?.status
  store.invalidate(artifactPath.value)
  await load()
  const newStatus = artifact.value?.status
  if (newStatus !== prevStatus) {
    if (newStatus === 'blocked') {
      ui.info('Status changed to blocked — open questions detected.')
    } else if (prevStatus === 'blocked') {
      ui.info('Blocked status cleared — open questions resolved.')
    } else if (newStatus === 'in-qa') {
      ui.info('QA run started — artifact is now in-qa')
    } else if (prevStatus === 'in-qa' && newStatus === 'approved') {
      isStale.value = false
      ui.success('QA run completed — artifact returned to approved')
    }
  }
  // Clear stale warning whenever artifact leaves in-qa
  if (prevStatus === 'in-qa' && newStatus !== 'in-qa') {
    isStale.value = false
  }
})

watch(artifactPath, load, { immediate: false })
onMounted(() => {
  load()
  if (graphStore.rawEdges.length === 0) {
    graphStore.fetchGraph(project.value)
  }
})
</script>

<template>
  <div class="editor-view">
    <!-- Top bar -->
    <div class="editor-topbar">
      <LineageBreadcrumb
        v-if="artifact"
        :project="project"
        :path="artifactPath"
        :lineage="artifact.lineage"
      />
      <div v-else class="crumb-back-wrap">
        <button class="crumb-back" @click="router.push(`/p/${project}/artifacts`)">← artifacts</button>
      </div>

      <div class="topbar-actions" v-if="artifact && !loading">
        <template v-if="!editing">
          <button
            v-if="isTestArtifact"
            class="btn-run-test"
            :disabled="runTestRunning"
            @click="runTest"
          >{{ runTestRunning ? 'Running…' : 'Run Test' }}</button>
          <button class="btn-ghost" @click="showRunAgent = true">Run Agent</button>
          <button
            class="btn-primary"
            :disabled="!!conflictLock"
            :title="conflictLock ? `Locked by ${conflictLock.holder}` : 'Edit artifact'"
            @click="enterEdit"
          >Edit</button>
        </template>
        <template v-else>
          <button class="btn-primary" :disabled="saving" @click="save">
            {{ saving ? 'Saving…' : 'Save' }}
          </button>
          <button class="btn-ghost" :disabled="saving" @click="cancelEdit">Cancel</button>
          <span class="shortcut-hint">Cmd+S to save</span>
        </template>
      </div>
    </div>

    <!-- Lock banner (locked by someone else) -->
    <LockBanner v-if="conflictLock" :lock="conflictLock" />

    <!-- Stale in-qa warning banner -->
    <div v-if="isStale && !editing" class="stale-qa-banner">
      This test has been in-qa for over 60 minutes — the QA run may be stuck.
    </div>

    <!-- External change banner -->
    <div v-if="hasExternalChange" class="external-change-banner">
      <span>This file was changed externally.</span>
      <button class="btn-link" @click="reloadFromDisk">Reload from disk</button>
      <button class="btn-link muted" @click="acknowledge">Keep editing</button>
    </div>

    <!-- Body -->
    <div v-if="loading" class="state-msg">Loading…</div>
    <div v-else-if="loadError" class="state-msg error">{{ loadError }}</div>
    <div v-else-if="!artifact" class="state-msg">Not found.</div>

    <!-- Read mode -->
    <div v-else-if="!editing" class="editor-body">
      <div class="editor-content">
        <h1 class="artifact-title">{{ artifact.title || artifact.slug }}</h1>
        <div v-if="artifact.status === 'blocked' && hasOpenQuestions" class="blocked-questions-banner">
          This artifact is blocked pending answers to open questions below.
          <a href="#open-questions" class="oq-link" @click.prevent="scrollToOpenQuestions">Jump to questions ↓</a>
        </div>
        <MarkdownPreview :html="artifact.body_html" :source="artifact.body" :project="project" />
      </div>
      <FrontmatterPanel
        :artifact="artifact"
        :project="project"
        :target-path="artifactPath"
        :edges="graphStore.rawEdges"
        @transitioned="(s) => { if (artifact) artifact = { ...artifact, status: s }; store.invalidate(artifactPath) }"
        @error="(msg) => ui.error(msg)"
      />
    </div>

    <!-- Edit mode: split pane -->
    <div v-else class="editor-split">
      <div class="split-editor">
        <MarkdownEditor v-model="editBody" @save="save" />
      </div>
      <div class="split-preview">
        <MarkdownPreview :source="editBody" :project="project" />
      </div>
      <FrontmatterEditor
        v-if="editFrontmatter"
        v-model="editFrontmatter"
        :project="project"
      />
    </div>
  </div>

  <RunAgentDialog
    v-if="showRunAgent && artifact"
    :project="project"
    :target-path="artifactPath"
    @started="showRunAgent = false"
    @cancel="showRunAgent = false"
  />
</template>

<style scoped>
.editor-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.editor-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg);
  flex-shrink: 0;
  gap: var(--space-4);
}
.crumb-back-wrap { font-size: var(--text-sm); }
.crumb-back {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-accent);
  font-size: var(--text-sm);
  padding: 0;
  font-family: inherit;
}
.crumb-back:hover { text-decoration: underline; }
.topbar-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
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
.btn-run-test {
  padding: var(--space-1) var(--space-4);
  background: #059669;
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-run-test:hover:not(:disabled) { opacity: 0.88; }
.btn-run-test:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-ghost {
  padding: var(--space-1) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-ghost:hover:not(:disabled) { background: var(--color-surface); color: var(--color-text); }
.btn-ghost:disabled { opacity: 0.5; cursor: not-allowed; }
.shortcut-hint {
  font-size: 11px;
  color: var(--color-text-muted);
}
.stale-qa-banner {
  display: flex;
  align-items: center;
  padding: var(--space-2) var(--space-6);
  background: #fffbeb;
  color: #92400e;
  font-size: var(--text-sm);
  font-weight: 500;
  border-bottom: 1px solid #fcd34d;
  flex-shrink: 0;
}
@media (prefers-color-scheme: dark) {
  .stale-qa-banner { background: #422006; color: #fcd34d; border-color: #92400e; }
}
.external-change-banner {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-6);
  background: #eff6ff;
  color: #1d4ed8;
  font-size: var(--text-sm);
  border-bottom: 1px solid #bfdbfe;
  flex-shrink: 0;
}
@media (prefers-color-scheme: dark) {
  .external-change-banner { background: #1e3a5f; color: #93c5fd; border-color: #1e40af; }
}
.btn-link {
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  color: inherit;
  font-size: inherit;
  font-family: inherit;
  font-weight: 600;
  text-decoration: underline;
}
.btn-link.muted { font-weight: normal; opacity: 0.7; }
.state-msg {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.state-msg.error { color: #dc2626; }

/* Read mode */
.editor-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}
.editor-content {
  flex: 1;
  padding: var(--space-8);
  overflow-y: auto;
}
.artifact-title {
  font-size: var(--text-2xl);
  font-weight: 700;
  margin: 0 0 var(--space-6);
  color: var(--color-text);
}

/* Blocked-on-questions banner */
.blocked-questions-banner {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
  padding: var(--space-3) var(--space-4);
  margin-bottom: var(--space-4);
  background: #fffbeb;
  color: #92400e;
  border: 1px solid #fcd34d;
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
}
@media (prefers-color-scheme: dark) {
  .blocked-questions-banner { background: #422006; color: #fcd34d; border-color: #92400e; }
}
.oq-link {
  color: inherit;
  font-weight: 700;
  white-space: nowrap;
}

/* Edit mode: split pane */
.editor-split {
  display: flex;
  flex: 1;
  overflow: hidden;
}
.split-editor {
  flex: 1;
  overflow: hidden;
  border-right: 1px solid var(--color-border);
}
.split-preview {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-6) var(--space-8);
}
</style>
