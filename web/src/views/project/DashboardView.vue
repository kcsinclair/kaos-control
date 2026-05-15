<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessageSquarePlus, Bug } from 'lucide-vue-next'
import DashboardGrid from '@/components/dashboard/DashboardGrid.vue'
import AgentRunningBanner from '@/components/dashboard/AgentRunningBanner.vue'
import BrainDumpModal from '@/components/idea/BrainDumpModal.vue'
import { useBrainDumpStore } from '@/stores/brainDump'
import { useUiStore } from '@/stores/ui'

const route = useRoute()
const router = useRouter()
const project = computed(() => route.params.project as string)

const brainDumpStore = useBrainDumpStore()
const ui = useUiStore()

const showBrainDump = ref(false)
const brainDumpType = ref<'idea' | 'defect'>('idea')
const triggerButtonEl = ref<HTMLButtonElement | null>(null)

function openBrainDump(type: 'idea' | 'defect', el: HTMLButtonElement) {
  brainDumpType.value = type
  triggerButtonEl.value = el
  brainDumpStore.reset()
  showBrainDump.value = true
}

function onBrainDumpClose() {
  showBrainDump.value = false
  nextTick(() => triggerButtonEl.value?.focus())
}

function onBrainDumpCreated(path: string) {
  showBrainDump.value = false
  ui.success('Artifact created!')
  router.push(`/p/${project.value}/artifacts/${path}`)
}
</script>

<template>
  <div class="dashboard-view">
    <header class="dashboard-header">
      <h2 class="dashboard-title">Dashboard</h2>
      <div class="header-actions">
        <button
          class="btn-new-idea"
          @click="openBrainDump('idea', $event.currentTarget as HTMLButtonElement)"
        >
          <MessageSquarePlus :size="15" />
          New Idea
        </button>
        <button
          class="btn-new-defect"
          @click="openBrainDump('defect', $event.currentTarget as HTMLButtonElement)"
        >
          <Bug :size="15" />
          New Defect
        </button>
      </div>
    </header>
    <AgentRunningBanner :project="project" />
    <DashboardGrid :project="project" />

    <BrainDumpModal
      v-if="showBrainDump"
      :project="project"
      :artifact-type="brainDumpType"
      @close="onBrainDumpClose"
      @created="onBrainDumpCreated"
    />
  </div>
</template>

<style scoped>
.dashboard-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  /* Prevent any widget from causing horizontal scroll */
  overflow-x: hidden;
  overflow-y: auto;
  padding: var(--space-6) var(--space-4);
  box-sizing: border-box;
  min-width: 0;
}

@media (max-width: 1023px) {
  .dashboard-view {
    padding: var(--space-4) var(--space-3);
  }
}

.dashboard-header {
  display: flex;
  align-items: center;
  margin-bottom: var(--space-4);
}

.dashboard-title {
  font-size: var(--text-xl);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.header-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.btn-new-defect {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-new-defect:hover { background: var(--color-surface); color: var(--color-text); }
.btn-new-defect:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 2px; }

.btn-new-idea {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-new-idea:hover { opacity: 0.88; }
.btn-new-idea:focus-visible { outline: 2px solid var(--color-accent); outline-offset: 2px; }
</style>
