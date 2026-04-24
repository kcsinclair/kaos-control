<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useArtifactsStore } from '@/stores/artifacts'
import { useWebSocket } from '@/composables/useWebSocket'
import LineageBreadcrumb from '@/components/artifact/LineageBreadcrumb.vue'
import FrontmatterPanel from '@/components/artifact/FrontmatterPanel.vue'
import MarkdownPreview from '@/components/artifact/MarkdownPreview.vue'
import type { ArtifactDetail, WsEvent } from '@/types/api'

const route = useRoute()
const router = useRouter()
const store = useArtifactsStore()

const project = computed(() => route.params.project as string)
const artifactPath = computed(() => {
  const m = route.params.pathMatch
  return Array.isArray(m) ? m.join('/') : (m as string)
})

const artifact = ref<ArtifactDetail | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

async function load() {
  if (!artifactPath.value) return
  loading.value = true
  error.value = null
  try {
    artifact.value = await store.fetchOne(project.value, artifactPath.value)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load artifact'
  } finally {
    loading.value = false
  }
}

// Reload when the artifact is re-indexed via WebSocket
useWebSocket(project.value, 'artifact.indexed', (e: WsEvent) => {
  if (e.payload?.path === artifactPath.value) {
    store.invalidate(artifactPath.value)
    load()
  }
})

watch(artifactPath, load, { immediate: false })
onMounted(load)
</script>

<template>
  <div class="editor-view">
    <div class="editor-topbar">
      <LineageBreadcrumb
        v-if="artifact"
        :project="project"
        :path="artifactPath"
        :lineage="artifact.lineage"
      />
      <div v-else class="breadcrumb-placeholder">
        <button class="crumb-back" @click="router.push(`/p/${project}/artifacts`)">← artifacts</button>
      </div>
    </div>

    <div v-if="loading" class="state-msg">Loading…</div>
    <div v-else-if="error" class="state-msg error">{{ error }}</div>
    <div v-else-if="!artifact" class="state-msg">Not found.</div>
    <div v-else class="editor-body">
      <div class="editor-content">
        <h1 class="artifact-title">{{ artifact.title || artifact.slug }}</h1>
        <MarkdownPreview :html="artifact.body_html" :source="artifact.body" :project="project" />
      </div>
      <FrontmatterPanel :artifact="artifact" />
    </div>
  </div>
</template>

<style scoped>
.editor-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.editor-topbar {
  padding: var(--space-3) var(--space-6);
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg);
  flex-shrink: 0;
}
.breadcrumb-placeholder {
  font-size: var(--text-sm);
}
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
.state-msg {
  padding: var(--space-8) var(--space-6);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.state-msg.error { color: #dc2626; }
.editor-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}
.editor-content {
  flex: 1;
  padding: var(--space-8) var(--space-8);
  overflow-y: auto;
}
.artifact-title {
  font-size: var(--text-2xl);
  font-weight: 700;
  margin: 0 0 var(--space-6);
  color: var(--color-text);
}
</style>
