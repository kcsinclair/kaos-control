<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
// Milestone 6: post-commit confirmation. Links to the promoted architecture,
// stack, summary, and ADR so the user can see exactly what was written, plus
// an entry point into the optional M7 scaffolding step. The backend already
// clears the saved resumable state on a successful commit; the wizard
// shell resets its own local state when this view unmounts.
import { useArchitectureWizardStore } from '@/stores/architectureWizard'

defineProps<{
  project: string
}>()

const emit = defineEmits<{
  scaffold: []
}>()

const store = useArchitectureWizardStore()
</script>

<template>
  <div class="success-step">
    <h2 class="success-title">Architecture selected</h2>
    <p v-if="store.commitResult" class="success-copy">
      Your choice has been written to the project.
    </p>

    <ul v-if="store.commitResult" class="success-links">
      <li>
        <router-link :to="`/p/${project}/artifacts/${store.commitResult.promoted_architecture}`">
          Promoted architecture
        </router-link>
      </li>
      <li>
        <router-link :to="`/p/${project}/artifacts/${store.commitResult.promoted_tech_stack}`">
          Promoted tech stack
        </router-link>
      </li>
      <li>
        <router-link :to="`/p/${project}/artifacts/${store.commitResult.summary_path}`">
          Architecture summary
        </router-link>
      </li>
      <li>
        <router-link :to="`/p/${project}/artifacts/${store.commitResult.adr_path}`">
          Decision record
        </router-link>
      </li>
      <li v-if="store.commitResult.superseded_adr_path">
        <router-link :to="`/p/${project}/artifacts/${store.commitResult.superseded_adr_path}`">
          Superseded decision record
        </router-link>
      </li>
    </ul>

    <p v-if="store.scaffoldSettled" class="success-copy scaffold-settled">
      Scaffolding step complete.
    </p>

    <div class="success-actions">
      <router-link :to="`/p/${project}/architecture/map`" class="btn-secondary">
        View relationship map
      </router-link>
      <button
        v-if="!store.scaffoldSettled"
        type="button"
        class="btn-primary"
        @click="emit('scaffold')"
      >
        Set up scaffolding
      </button>
    </div>
  </div>
</template>

<style scoped>
.success-step {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.success-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
}
.success-copy {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.success-copy.scaffold-settled {
  font-weight: 600;
  color: var(--color-accent);
}
.success-links {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: var(--text-sm);
}
.success-links a {
  color: var(--color-accent);
}
.success-actions {
  display: flex;
  gap: var(--space-3);
}
.btn-primary {
  padding: var(--space-2) var(--space-5);
  background: var(--color-accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-primary:hover { opacity: 0.88; }
.btn-secondary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  text-decoration: none;
}
.btn-secondary:hover { background: var(--color-border); }
</style>
