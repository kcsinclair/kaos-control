<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { ApiError } from '@/api/client'
import AppHeader from '@/components/layout/AppHeader.vue'

const router = useRouter()
const projectStore = useProjectStore()
const authStore = useAuthStore()
const ui = useUiStore()

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
</script>

<template>
  <div class="picker-page">
    <AppHeader />
    <main class="picker-main">
      <div class="picker-content">
        <h2 class="picker-heading">Projects</h2>

        <div v-if="projectStore.loading" class="picker-empty">Loading…</div>

        <div v-else-if="projectStore.projects.length === 0" class="picker-empty">
          No projects registered. Add a project registration file to your projects directory.
        </div>

        <ul v-else class="project-list">
          <li
            v-for="p in projectStore.projects"
            :key="p.name"
            class="project-item"
            @click="openProject(p.name)"
          >
            <div class="project-name">{{ p.name }}</div>
            <div v-if="p.description" class="project-desc">{{ p.description }}</div>
            <div class="project-roles">
              <span
                v-for="role in authStore.rolesForProject(p.name)"
                :key="role"
                class="role-chip"
              >{{ role }}</span>
            </div>
          </li>
        </ul>
      </div>
    </main>
  </div>
</template>

<style scoped>
.picker-page {
  min-height: 100vh;
  background: var(--color-bg);
  display: flex;
  flex-direction: column;
}
.picker-main {
  flex: 1;
  display: flex;
  justify-content: center;
  padding: var(--space-8) var(--space-4);
}
.picker-content {
  width: 100%;
  max-width: 680px;
}
.picker-heading {
  font-size: var(--text-xl);
  font-weight: 600;
  margin: 0 0 var(--space-4);
  color: var(--color-text);
}
.picker-empty {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  padding: var(--space-6) 0;
}
.project-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.project-item {
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.project-item:hover {
  border-color: var(--color-accent);
  box-shadow: var(--shadow-sm);
}
.project-name {
  font-weight: 600;
  color: var(--color-text);
  font-size: var(--text-base);
}
.project-desc {
  margin-top: var(--space-1);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.project-roles {
  margin-top: var(--space-2);
  display: flex;
  gap: var(--space-1);
  flex-wrap: wrap;
}
.role-chip {
  display: inline-block;
  padding: 2px 8px;
  font-size: var(--text-xs);
  background: var(--color-accent-subtle);
  color: var(--color-accent);
  border-radius: var(--radius-full);
  font-weight: 500;
}
</style>
