<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'

const props = defineProps<{
  project: string
  path: string
  lineage: string
}>()

const router = useRouter()

const segments = computed(() => {
  const parts = props.path.split('/')
  const result: { label: string; path: string }[] = []
  for (let i = 0; i < parts.length; i++) {
    result.push({
      label: parts[i],
      path: parts.slice(0, i + 1).join('/'),
    })
  }
  return result
})

function toArtifact(path: string) {
  router.push(`/p/${props.project}/artifacts/${path}`)
}
</script>

<template>
  <nav class="breadcrumb" aria-label="Artifact path">
    <button class="crumb-link" @click="router.push(`/p/${project}/artifacts`)">
      artifacts
    </button>
    <span class="sep">/</span>
    <template v-for="(seg, i) in segments" :key="seg.path">
      <button
        v-if="i < segments.length - 1"
        class="crumb-link"
        @click="toArtifact(seg.path)"
      >{{ seg.label }}</button>
      <span v-else class="crumb-current">{{ seg.label }}</span>
      <span v-if="i < segments.length - 1" class="sep">/</span>
    </template>
  </nav>
</template>

<style scoped>
.breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  flex-wrap: wrap;
}
.crumb-link {
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  color: var(--color-accent);
  font-size: inherit;
  font-family: inherit;
}
.crumb-link:hover {
  text-decoration: underline;
}
.crumb-current {
  color: var(--color-text);
  font-weight: 500;
}
.sep {
  color: var(--color-border);
}
</style>
