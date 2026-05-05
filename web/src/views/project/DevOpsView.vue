<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()

const project = route.params.project as string

const hasAccess = computed(() => {
  const roles = auth.rolesForProject(project)
  return roles.includes('product-owner') || roles.includes('devops')
})
</script>

<template>
  <div class="devops-view">
    <div v-if="!hasAccess" class="access-denied">
      <p class="access-denied__msg">You don't have permission to access DevOps pipelines.</p>
      <p class="access-denied__hint">Required role: <code>product-owner</code> or <code>devops</code>.</p>
    </div>
    <template v-else>
      <div class="devops-header">
        <h2 class="devops-title">DevOps Pipelines</h2>
      </div>
      <div class="devops-content">
        <p class="placeholder-msg">Loading pipelines…</p>
      </div>
    </template>
  </div>
</template>

<style scoped>
.devops-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.devops-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.devops-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}
.devops-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-6);
}
.placeholder-msg {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.access-denied {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: var(--space-2);
}
.access-denied__msg {
  font-size: var(--text-base);
  color: var(--color-text);
  margin: 0;
}
.access-denied__hint {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  margin: 0;
}
</style>
