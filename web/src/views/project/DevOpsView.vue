<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useDevOpsStore } from '@/stores/devops'
import PipelineCard from '@/components/devops/PipelineCard.vue'

const route = useRoute()
const auth = useAuthStore()
const devops = useDevOpsStore()

const project = route.params.project as string

const hasAccess = computed(() => {
  const roles = auth.rolesForProject(project)
  return roles.includes('product-owner') || roles.includes('devops')
})

const columnOrder = ['build', 'deploy', 'release']

const orderedTypes = computed((): string[] => {
  const types = Object.keys(devops.pipelinesByType)
  const known = columnOrder.filter((t) => types.includes(t))
  const dynamic = types.filter((t) => !columnOrder.includes(t)).sort()
  return [...known, ...dynamic]
})

onMounted(() => {
  if (hasAccess.value) {
    devops.fetchPipelines(project)
  }
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
        <div v-if="devops.loading" class="state-msg">Loading pipelines…</div>
        <div v-else-if="devops.loadError" class="state-msg state-msg--error">{{ devops.loadError }}</div>
        <div v-else-if="devops.pipelines.length === 0" class="state-msg">No pipelines found. Add YAML files to <code>lifecycle/devops/</code>.</div>
        <div v-else class="columns">
          <div
            v-for="type in orderedTypes"
            :key="type"
            class="column"
          >
            <h3 class="column-header">{{ type.charAt(0).toUpperCase() + type.slice(1) }}</h3>
            <div class="card-list">
              <PipelineCard
                v-for="pipeline in devops.pipelinesByType[type]"
                :key="pipeline.slug"
                :pipeline="pipeline"
              />
            </div>
          </div>
        </div>
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
.state-msg {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.state-msg--error {
  color: var(--color-error);
}
.columns {
  display: flex;
  gap: var(--space-6);
  align-items: flex-start;
}
.column {
  flex: 1;
  min-width: 220px;
  max-width: 380px;
}
.column-header {
  font-size: var(--text-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-text-muted);
  margin: 0 0 var(--space-3) 0;
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--color-border);
}
.card-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
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
