<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { X } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { getConfigHealth } from '@/api/config'
import type { RepairNote } from '@/types/api'

const props = defineProps<{ project: string }>()

const authStore = useAuthStore()
const repairs = ref<RepairNote[]>([])
const dismissed = ref(false)

const isAdmin = computed(() => {
  const roles = authStore.rolesForProject(props.project)
  return roles.includes('admin') || roles.includes('product-owner')
})

async function fetchHealth() {
  if (!isAdmin.value) {
    repairs.value = []
    return
  }
  try {
    const res = await getConfigHealth(props.project)
    repairs.value = res.repairs
  } catch {
    // Non-blocking hint — silently skip on failure.
    repairs.value = []
  }
}

watch(
  () => props.project,
  () => {
    dismissed.value = false
    void fetchHealth()
  },
  { immediate: true },
)

const templateSummary = computed(() =>
  repairs.value.map((r) => `${r.template_key} (${r.agent})`).join(', '),
)

const visible = computed(() => isAdmin.value && repairs.value.length > 0 && !dismissed.value)
</script>

<template>
  <div v-if="visible" class="config-health-banner" role="status">
    <span class="chb-text">
      kaos-control auto-filled {{ repairs.length }} missing agent template{{ repairs.length === 1 ? '' : 's' }}
      at startup: {{ templateSummary }}. Add
      {{ repairs.length === 1 ? 'it' : 'them' }} to lifecycle/config.yaml to make {{ repairs.length === 1 ? 'it' : 'this' }} permanent.
    </span>
    <button class="chb-dismiss" aria-label="Dismiss config-health hint" @click="dismissed = true">
      <X :size="14" />
    </button>
  </div>
</template>

<style scoped>
.config-health-banner {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-6);
  background: #eff6ff;
  color: #1e40af;
  font-size: var(--text-sm);
  border-bottom: 1px solid #bfdbfe;
  flex-shrink: 0;
}
@media (prefers-color-scheme: dark) {
  .config-health-banner { background: #172554; color: #93c5fd; border-color: #1e3a8a; }
}
.chb-text {
  flex: 1;
}
.chb-dismiss {
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  color: inherit;
  opacity: 0.7;
  cursor: pointer;
  padding: 0;
  flex-shrink: 0;
}
.chb-dismiss:hover {
  opacity: 1;
}
</style>
