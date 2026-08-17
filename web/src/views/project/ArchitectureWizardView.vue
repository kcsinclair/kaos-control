<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import WizardStepper from '@/components/architecture/WizardStepper.vue'
import PriorRunGate from '@/components/architecture/PriorRunGate.vue'
import PathChoiceStep from '@/components/architecture/PathChoiceStep.vue'
import BrowseCatalogStep from '@/components/architecture/BrowseCatalogStep.vue'
import StackChoiceStep from '@/components/architecture/StackChoiceStep.vue'
import type { WizardStepperStep } from '@/components/architecture/WizardStepper.vue'
import type { CatalogItem } from '@/types/api'

const route = useRoute()
const router = useRouter()
const store = useArchitectureWizardStore()

const project = computed(() => route.params.project as string)

// Step sequence branches on the chosen path (FR-4): Browse skips straight
// from the catalog to the stack picker; Guided runs the question set then
// shows a transparent recommendation before the same stack picker.
const STEPS_GUIDED: WizardStepperStep[] = [
  { key: 'path', label: 'Path' },
  { key: 'questions', label: 'Questions' },
  { key: 'recommend', label: 'Recommendation' },
  { key: 'stack', label: 'Choose stack' },
  { key: 'confirm', label: 'Confirm' },
  { key: 'done', label: 'Done' },
]
const STEPS_BROWSE: WizardStepperStep[] = [
  { key: 'path', label: 'Path' },
  { key: 'browse', label: 'Browse' },
  { key: 'stack', label: 'Choose stack' },
  { key: 'confirm', label: 'Confirm' },
  { key: 'done', label: 'Done' },
]
const STEPS = computed(() => (store.path === 'browse' ? STEPS_BROWSE : STEPS_GUIDED))

// Whether the user has explicitly acknowledged an already-run wizard (FR-3) —
// separate from store.priorRun itself, which stays populated so the gate can
// re-render if the user navigates back to it.
const priorRunAcknowledged = ref(false)

const showPriorRunGate = computed(
  () => store.priorRun?.detected === true && !priorRunAcknowledged.value,
)

const currentStepKey = computed(() =>
  STEPS.value.some((s) => s.key === store.step) ? store.step : 'path',
)
const currentStepIndex = computed(() =>
  STEPS.value.findIndex((s) => s.key === currentStepKey.value),
)

onMounted(() => {
  void store.start(project.value)
})

function onPriorRunContinue(): void {
  priorRunAcknowledged.value = true
}

function exitWizard(): void {
  store.reset()
  void router.push(`/p/${encodeURIComponent(project.value)}/architecture/map`)
}

function goBack(): void {
  if (currentStepIndex.value <= 0) {
    exitWizard()
    return
  }
  store.step = STEPS.value[currentStepIndex.value - 1].key
}

function goNext(): void {
  if (currentStepIndex.value >= STEPS.value.length - 1) return
  store.step = STEPS.value[currentStepIndex.value + 1].key
}

function onChoosePath(path: 'browse' | 'guided'): void {
  store.setPath(path)
  store.persistState(project.value)
  store.step = path === 'browse' ? 'browse' : 'questions'
}

function onArchitectureChosen(item: CatalogItem): void {
  store.chooseArchitecture(item)
  store.persistState(project.value)
  store.step = 'stack'
}

function onStackChosen(item: CatalogItem): void {
  store.chooseStack(item)
  store.persistState(project.value)
  store.step = 'confirm'
}

// The footer Next button is a fallback for re-advancing through
// already-completed steps (e.g. after Back) — each step's own primary
// action is what normally advances the wizard.
const canAdvance = computed(() => {
  switch (currentStepKey.value) {
    case 'path':
      return store.isPathChosen
    case 'browse':
      return store.isArchitectureChosen
    case 'questions':
      return store.recommendations.length > 0
    case 'recommend':
      return store.isArchitectureChosen
    case 'stack':
      return store.isStackChosen
    default:
      return false
  }
})
</script>

<template>
  <div class="wizard-view">
    <h1 class="wizard-title">Architecture Wizard</h1>

    <div v-if="store.error" class="error-banner">{{ store.error }}</div>

    <div v-if="store.loading && !store.priorRunResolved" class="wizard-loading">Loading…</div>

    <template v-else>
      <PriorRunGate
        v-if="showPriorRunGate"
        :prior-run="store.priorRun!"
        @continue="onPriorRunContinue"
        @exit="exitWizard"
      />

      <template v-else>
        <WizardStepper :steps="STEPS" :current-key="currentStepKey" />

        <div class="wizard-step-body">
          <PathChoiceStep v-if="currentStepKey === 'path'" @choose="onChoosePath" />

          <BrowseCatalogStep
            v-else-if="currentStepKey === 'browse'"
            :project="project"
            @chosen="onArchitectureChosen"
          />

          <StackChoiceStep
            v-else-if="currentStepKey === 'stack'"
            :project="project"
            @chosen="onStackChosen"
          />

          <!-- Guided questions/recommendation and confirm/done are filled in by
               later milestones — this shell only hosts navigation. -->
          <p v-else class="step-placeholder">
            This step isn't implemented yet — see
            lifecycle/frontend-plans/onboarding-architecture-selection-4-fe.md.
          </p>
        </div>

        <div class="wizard-footer">
          <button type="button" class="btn-secondary" @click="exitWizard">Cancel</button>
          <div class="wizard-footer-nav">
            <button type="button" class="btn-secondary" @click="goBack">Back</button>
            <button
              type="button"
              class="btn-primary"
              :disabled="currentStepIndex >= STEPS.length - 1 || !canAdvance"
              @click="goNext"
            >Next</button>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.wizard-view {
  max-width: 800px;
  margin: 0 auto;
  padding: var(--space-6) var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.wizard-title {
  margin: 0;
  font-size: var(--text-xl);
  font-weight: 700;
  color: var(--color-text);
}
.error-banner {
  padding: var(--space-3);
  background: #fee2e2;
  border: 1px solid #fca5a5;
  border-radius: var(--radius-md);
  color: #991b1b;
  font-size: var(--text-sm);
}
.wizard-loading {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.wizard-step-body {
  min-height: 200px;
  padding: var(--space-6);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.step-placeholder {
  margin: 0;
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.wizard-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.wizard-footer-nav {
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
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-primary:not(:disabled):hover { opacity: 0.88; }
.btn-secondary {
  padding: var(--space-2) var(--space-4);
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
}
.btn-secondary:hover { background: var(--color-border); }
</style>
