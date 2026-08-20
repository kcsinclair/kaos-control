<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useArtifactsStore } from '@/stores/artifacts'
import MarkdownPreview from '@/components/artifact/MarkdownPreview.vue'
import type { ArtifactDetail, OverviewItem } from '@/types/api'

// Tech-stack panel + mapping (FR-3). Per OQ-3 the architecture↔stack mapping
// is the hard `related_to` references already present in the promoted
// artifacts' frontmatter — no new mapping field. See [[architecture-overview-view]].
const props = defineProps<{
  project: string
  stack: OverviewItem | null
  architecture: OverviewItem | null
}>()

const store = useArtifactsStore()
const stackDetail = ref<ArtifactDetail | null>(null)
const archDetail = ref<ArtifactDetail | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

// lifecycle/architecture/*/related_to entries are relative to lifecycle/
// (e.g. "architecture/tech-stacks/go-vue.md") — normalise to a repo-relative
// path matching OverviewItem.path / the artifact-editor route.
function resolveRelatedPath(ref: string): string {
  let target = ref.trim()
  if (!target.startsWith('lifecycle/')) target = `lifecycle/${target}`
  if (!target.endsWith('.md')) target += '.md'
  return target
}

const relatedRefs = computed(() => {
  const raw = [
    ...(archDetail.value?.frontmatter?.related_to ?? []),
    ...(stackDetail.value?.frontmatter?.related_to ?? []),
  ]
  const seen = new Set<string>()
  const out: { path: string; label: string }[] = []
  for (const r of raw) {
    const path = resolveRelatedPath(r)
    if (seen.has(path)) continue
    seen.add(path)
    out.push({ path, label: path.split('/').pop() ?? path })
  }
  return out
})

async function load() {
  archDetail.value = null
  stackDetail.value = null
  error.value = null
  const stackPath = props.stack?.path
  const archPath = props.architecture?.path
  if (!stackPath && !archPath) return
  loading.value = true
  try {
    const [s, a] = await Promise.all([
      stackPath ? store.fetchOne(props.project, stackPath) : Promise.resolve(null),
      archPath ? store.fetchOne(props.project, archPath) : Promise.resolve(null),
    ])
    stackDetail.value = s
    archDetail.value = a
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load the tech stack.'
  } finally {
    loading.value = false
  }
}

watch(() => [props.stack?.path, props.architecture?.path], load, { immediate: true })
</script>

<template>
  <section class="overview-panel">
    <header class="panel-header">
      <h3 class="panel-title">Tech stack</h3>
    </header>

    <p v-if="!stack" class="panel-empty">No tech stack has been chosen yet.</p>
    <template v-else>
      <router-link :to="`/p/${project}/artifacts/${stack.path}`" class="panel-link-title">
        {{ stack.title }}
      </router-link>
      <p v-if="stackDetail?.frontmatter?.summary" class="panel-summary">{{ stackDetail.frontmatter.summary }}</p>

      <div v-if="loading" class="panel-loading">Loading…</div>
      <p v-else-if="error" class="panel-error">{{ error }}</p>
      <MarkdownPreview v-else-if="stackDetail" :source="stackDetail.body" :project="project" />

      <div v-if="relatedRefs.length" class="related-refs">
        <h4 class="related-refs-title">Architecture ↔ stack references</h4>
        <ul class="related-refs-list">
          <li v-for="ref in relatedRefs" :key="ref.path">
            <router-link :to="`/p/${project}/artifacts/${ref.path}`">{{ ref.label }}</router-link>
          </li>
        </ul>
      </div>
    </template>
  </section>
</template>

<style scoped>
.overview-panel {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
}
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-3);
}
.panel-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.panel-link-title {
  display: block;
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--color-text);
  text-decoration: none;
  margin-bottom: var(--space-2);
}
.panel-link-title:hover {
  color: var(--color-accent);
  text-decoration: underline;
}
.panel-summary {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  margin: 0 0 var(--space-3);
}
.panel-loading,
.panel-error {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.panel-error {
  color: var(--color-error);
}
.panel-empty {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
  margin: 0;
}
.related-refs {
  margin-top: var(--space-4);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}
.related-refs-title {
  font-size: var(--text-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  margin: 0 0 var(--space-2);
}
.related-refs-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.related-refs-list a {
  color: var(--color-accent);
  text-decoration: none;
  font-size: var(--text-sm);
}
.related-refs-list a:hover {
  text-decoration: underline;
}
</style>
