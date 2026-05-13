<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { watch } from 'vue'
import { GitBranch } from 'lucide-vue-next'
import { useGitStatusStore } from '@/stores/gitStatus'
import { useWebSocket } from '@/composables/useWebSocket'
import type { WsEvent } from '@/types/api'
import type { GitStatusResponse } from '@/types/api'

const props = defineProps<{
  project: string
  collapsed: boolean
}>()

const gitStatusStore = useGitStatusStore()

// Fetch on mount and whenever the project changes
gitStatusStore.fetch(props.project)
watch(
  () => props.project,
  (p) => { if (p) gitStatusStore.fetch(p) },
)

// Subscribe to live WebSocket updates
useWebSocket(props.project, 'git.status', (e: WsEvent) => {
  gitStatusStore.applyWsEvent(e.payload as unknown as GitStatusResponse)
})

function abbreviateSha(sha: string): string {
  return sha.slice(0, 7)
}

function firstLine(msg: string): string {
  return msg.split('\n')[0]
}
</script>

<template>
  <div
    v-if="gitStatusStore.available"
    class="git-status-bar"
    :class="{ 'git-status-bar--collapsed': collapsed }"
    role="status"
    aria-label="Git repository status"
    tabindex="0"
  >
    <!-- Collapsed: icon-only with dirty dot overlay -->
    <div v-if="collapsed" class="git-icon-wrap">
      <GitBranch :size="18" />
      <span
        class="git-dirty-dot"
        :class="gitStatusStore.dirty ? 'git-dirty-dot--dirty' : 'git-dirty-dot--clean'"
        :aria-label="gitStatusStore.dirty ? 'Working tree has uncommitted changes' : 'Working tree is clean'"
      ></span>
    </div>

    <!-- Expanded: full panel -->
    <template v-else>
      <div class="git-branch-row">
        <GitBranch :size="14" class="git-branch-icon" />
        <span class="git-branch-name" :title="gitStatusStore.branch">{{ gitStatusStore.branch }}</span>
        <span
          class="git-dirty-indicator"
          :class="gitStatusStore.dirty ? 'git-dirty-indicator--dirty' : 'git-dirty-indicator--clean'"
          :aria-label="gitStatusStore.dirty ? 'Working tree has uncommitted changes' : 'Working tree is clean'"
        >{{ gitStatusStore.dirty ? 'modified' : 'clean' }}</span>
      </div>
      <div v-if="gitStatusStore.headSha" class="git-commit-row">
        <span class="git-sha">{{ abbreviateSha(gitStatusStore.headSha) }}</span>
        <span class="git-commit-msg" :title="gitStatusStore.headMessage">{{ firstLine(gitStatusStore.headMessage) }}</span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.git-status-bar {
  padding: var(--space-2) var(--space-4);
  border-top: 1px solid var(--color-border-dark);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  outline: none;
}

.git-status-bar:focus-visible {
  outline: 2px solid var(--color-sidebar-active);
  outline-offset: -2px;
}

.git-status-bar--collapsed {
  padding: var(--space-2) 0;
  align-items: center;
}

/* Collapsed icon with dot */
.git-icon-wrap {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  color: var(--color-sidebar-text-muted);
}

.git-dirty-dot {
  position: absolute;
  top: -3px;
  right: -3px;
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
}

.git-dirty-dot--dirty {
  background: var(--color-warning);
}

.git-dirty-dot--clean {
  background: var(--color-success);
}

/* Expanded branch row */
.git-branch-row {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  overflow: hidden;
}

.git-branch-icon {
  flex-shrink: 0;
  color: var(--color-sidebar-text-muted);
}

.git-branch-name {
  flex: 1;
  font-size: var(--text-xs);
  font-weight: 600;
  color: var(--color-sidebar-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.git-dirty-indicator {
  flex-shrink: 0;
  font-size: var(--text-xs);
  font-weight: 500;
  border-radius: var(--radius-sm);
  padding: 0 var(--space-1);
  line-height: 1.4;
}

.git-dirty-indicator--dirty {
  color: var(--color-warning);
}

.git-dirty-indicator--clean {
  color: var(--color-success);
}

/* Expanded commit row */
.git-commit-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  overflow: hidden;
}

.git-sha {
  flex-shrink: 0;
  font-size: var(--text-xs);
  font-family: monospace;
  color: var(--color-sidebar-text-muted);
}

.git-commit-msg {
  flex: 1;
  font-size: var(--text-xs);
  color: var(--color-sidebar-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
