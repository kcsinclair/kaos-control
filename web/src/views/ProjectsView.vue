<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import { ApiError } from '@/api/client'
import AppHeader from '@/components/layout/AppHeader.vue'
import CreateProjectModal from '@/components/project/CreateProjectModal.vue'
import EditProjectModal from '@/components/project/EditProjectModal.vue'
import DeleteProjectModal from '@/components/project/DeleteProjectModal.vue'
import InitProjectModal from '@/components/project/InitProjectModal.vue'
import type { ProjectSummary } from '@/types/api'

const router = useRouter()
const projectStore = useProjectStore()
const ui = useUiStore()

const showCreate = ref(false)
const editTarget = ref<ProjectSummary | null>(null)
const deleteTarget = ref<ProjectSummary | null>(null)
const initTarget = ref<ProjectSummary | null>(null)

onMounted(async () => {
  try {
    await projectStore.fetchProjects()
  } catch (err) {
    if (err instanceof ApiError) {
      ui.error(err.message)
    }
  }
})

function openProject(name: string) {
  projectStore.setCurrent(name)
  router.push(`/p/${encodeURIComponent(name)}`)
}

function onCreated() {
  showCreate.value = false
}

function onUpdated() {
  editTarget.value = null
}

async function onDeleted() {
  const wasActive = deleteTarget.value?.name === projectStore.current?.name
  deleteTarget.value = null
  if (wasActive) {
    await router.push('/projects')
  }
}

function onInitialised() {
  initTarget.value = null
}
</script>

<template>
  <div class="projects-page">
    <AppHeader />
    <main class="projects-main">
      <div class="projects-content">
        <div class="projects-header">
          <h2 class="projects-heading">Projects</h2>
          <button class="btn-primary" @click="showCreate = true">New Project</button>
        </div>

        <div v-if="projectStore.loading" class="projects-empty">Loading…</div>

        <div v-else-if="projectStore.projects.length === 0" class="projects-empty">
          No projects registered. Click "New Project" to add one.
        </div>

        <div v-else class="projects-table-wrap">
          <table class="projects-table">
            <thead>
              <tr>
                <th>Name</th>
                <th class="col-desc">Description</th>
                <th class="col-owner">Owner</th>
                <th class="col-path">Path</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in projectStore.projects" :key="p.name">
                <td>
                  <a
                    class="project-name-link"
                    href="#"
                    @click.prevent="openProject(p.name)"
                  >{{ p.name }}</a>
                </td>
                <td class="col-desc td-muted">{{ p.description || '—' }}</td>
                <td class="col-owner td-muted">{{ p.owner || '—' }}</td>
                <td class="col-path">
                  <span class="path-text" :title="p.path">{{ p.path }}</span>
                </td>
                <td>
                  <span
                    class="status-badge"
                    :class="p.initialised ? 'status-badge--ok' : 'status-badge--warn'"
                  >{{ p.initialised ? 'Initialised' : 'Not initialised' }}</span>
                </td>
                <td>
                  <div class="row-actions">
                    <button class="btn-action" @click="editTarget = p">Edit</button>
                    <button
                      v-if="!p.initialised"
                      class="btn-action btn-action--init"
                      @click="initTarget = p"
                    >Initialise</button>
                    <button class="btn-action btn-action--danger" @click="deleteTarget = p">Delete</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </main>

    <!-- Modals -->
    <CreateProjectModal
      v-if="showCreate"
      @created="onCreated"
      @close="showCreate = false"
    />

    <EditProjectModal
      v-if="editTarget"
      :project="editTarget"
      @updated="onUpdated"
      @close="editTarget = null"
    />

    <DeleteProjectModal
      v-if="deleteTarget"
      :project="deleteTarget"
      @confirmed="onDeleted"
      @close="deleteTarget = null"
    />

    <InitProjectModal
      v-if="initTarget"
      :project="initTarget"
      @initialised="onInitialised"
      @close="initTarget = null"
    />
  </div>
</template>

<style scoped>
.projects-page {
  min-height: 100vh;
  background: var(--color-bg);
  display: flex;
  flex-direction: column;
}
.projects-main {
  flex: 1;
  padding: var(--space-6) var(--space-4);
}
.projects-content {
  width: 100%;
  max-width: 1100px;
  margin: 0 auto;
}
.projects-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-5);
  flex-wrap: wrap;
  gap: var(--space-3);
}
.projects-heading {
  font-size: var(--text-xl);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.btn-primary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}
.btn-primary:hover { opacity: 0.88; }
.projects-empty {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  padding: var(--space-6) 0;
}
.projects-table-wrap {
  overflow-x: auto;
}
.projects-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
.projects-table th {
  text-align: left;
  padding: var(--space-2) var(--space-3);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
  border-bottom: 1px solid var(--color-border);
  white-space: nowrap;
}
.projects-table td {
  padding: var(--space-3) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text);
  vertical-align: middle;
}
.projects-table tbody tr:hover td {
  background: var(--color-surface);
}
.project-name-link {
  font-weight: 600;
  color: var(--color-accent);
  text-decoration: none;
}
.project-name-link:hover { text-decoration: underline; }
.td-muted {
  color: var(--color-text-muted);
}
.col-desc { max-width: 220px; }
.col-owner { max-width: 140px; }
.col-path { max-width: 240px; }
.path-text {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-text-muted);
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
}
.status-badge--ok {
  background: #d1fae5;
  color: #065f46;
}
.status-badge--warn {
  background: #fef3c7;
  color: #92400e;
}
.row-actions {
  display: flex;
  gap: var(--space-2);
  flex-wrap: nowrap;
}
.btn-action {
  padding: var(--space-1) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: var(--text-xs);
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s, border-color 0.15s;
}
.btn-action:hover { background: var(--color-border); }
.btn-action--init {
  border-color: #f59e0b;
  color: #92400e;
}
.btn-action--init:hover { background: #fef3c7; }
.btn-action--danger {
  border-color: #fca5a5;
  color: #991b1b;
}
.btn-action--danger:hover { background: #fee2e2; }

@media (max-width: 768px) {
  .col-desc, .col-owner { display: none; }
  .projects-table th.col-desc,
  .projects-table th.col-owner { display: none; }
}
</style>
