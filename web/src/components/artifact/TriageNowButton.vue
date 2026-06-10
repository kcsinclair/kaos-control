<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Wand2 } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { triageIdea } from '@/api/ideas'
import { ApiError } from '@/api/client'
import type { ArtifactDetail } from '@/types/api'

const PERMITTED_ROLES = ['product-owner', 'analyst', 'reviewer']

const props = defineProps<{
  artifact: ArtifactDetail
  project: string
}>()

const emit = defineEmits<{ 'triage-started': [runId: string] }>()

const authStore = useAuthStore()
const ui = useUiStore()

const loading = ref(false)
const inlineError = ref<string | null>(null)

const visible = computed(() => {
  if (props.artifact?.type !== 'idea') return false
  if (props.artifact?.status !== 'raw') return false
  const roles = authStore.rolesForProject(props.project)
  return PERMITTED_ROLES.some((r) => roles.includes(r))
})

async function handleClick() {
  if (loading.value) return
  loading.value = true
  inlineError.value = null
  try {
    const res = await triageIdea(props.project, props.artifact.lineage)
    ui.success('Triage started')
    emit('triage-started', res.run_id)
  } catch (e: unknown) {
    if (e instanceof ApiError) {
      inlineError.value = `Cannot triage: ${e.code}`
    } else {
      inlineError.value = e instanceof Error ? e.message : 'Failed to start triage'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <template v-if="visible">
    <button
      class="btn-triage"
      :disabled="loading"
      :title="loading ? 'Triage in progress…' : 'Triage this idea now'"
      @click="handleClick"
    >
      <Wand2 :size="14" />
      {{ loading ? 'Triaging…' : 'Triage now' }}
    </button>
    <span v-if="inlineError" class="triage-error">{{ inlineError }}</span>
  </template>
</template>

<style scoped>
.btn-triage {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}
.btn-triage:hover:not(:disabled) {
  background: var(--color-surface);
  color: var(--color-text);
}
.btn-triage:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.triage-error {
  font-size: var(--text-sm);
  color: var(--color-error, #dc2626);
}
</style>
