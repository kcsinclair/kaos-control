<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
// Milestone 4 (FR-7, FR-8, NFR-5): the ≤10-question guided flow. Every
// question is skippable, with a "decide for me" affordance alongside plain
// Skip for less-technical users — both omit the question from the answer
// payload, letting the backend fall back to its defaults. Progresses
// through the backend-supplied question set, then hands off to the
// recommendation step.
import { computed, watch, onMounted } from 'vue'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'

const props = defineProps<{
  project: string
}>()

const emit = defineEmits<{
  complete: []
  'browse-anyway': []
}>()

const store = useArchitectureWizardStore()

const totalCount = computed(() => store.questions.length)
const positionLabel = computed(
  () => `Question ${Math.min(store.answeredQuestionCount + 1, totalCount.value)} of ${totalCount.value}`,
)

async function checkComplete(): Promise<void> {
  if (totalCount.value > 0 && store.currentQuestion === null) {
    await store.fetchRecommendations(props.project)
    emit('complete')
  }
}

function choose(value: string): void {
  const done = store.answerCurrentQuestion(props.project, value)
  if (done) void checkComplete()
}

function skipQuestion(): void {
  const done = store.skipCurrentQuestion(props.project)
  if (done) void checkComplete()
}

onMounted(checkComplete)
watch(() => store.questions, checkComplete)
</script>

<template>
  <div class="guided-step">
    <div class="guided-header">
      <span v-if="store.currentQuestion" class="guided-progress">{{ positionLabel }}</span>
      <button type="button" class="show-everything-btn" @click="emit('browse-anyway')">
        Show me everything anyway
      </button>
    </div>

    <div v-if="store.loading" class="guided-state" role="status" aria-live="polite">
      Computing your recommendation…
    </div>

    <div v-else-if="store.currentQuestion" class="guided-question" role="group" :aria-label="store.currentQuestion.prompt">
      <h2 class="guided-prompt">{{ store.currentQuestion.prompt }}</h2>
      <div class="guided-options">
        <button
          v-for="o in store.currentQuestion.options"
          :key="o.value"
          type="button"
          class="option-btn"
          @click="choose(o.value)"
        >{{ o.label }}</button>
      </div>
      <div class="guided-defaults">
        <button type="button" class="skip-btn" @click="skipQuestion">Skip</button>
        <button type="button" class="decide-btn" @click="skipQuestion">
          Not sure — decide for me
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.guided-step {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.guided-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}
.guided-progress {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text-muted);
}
.show-everything-btn {
  background: none;
  border: none;
  padding: 0;
  color: var(--color-accent);
  font-size: var(--text-sm);
  font-weight: 500;
  cursor: pointer;
  text-decoration: underline;
}
.guided-state {
  padding: var(--space-4);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.guided-question {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.guided-prompt {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
}
.guided-options {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
}
.option-btn {
  padding: var(--space-2) var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
  cursor: pointer;
}
.option-btn:hover,
.option-btn:focus-visible {
  border-color: var(--color-accent);
  color: var(--color-accent);
}
.guided-defaults {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}
.skip-btn,
.decide-btn {
  background: none;
  border: none;
  padding: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
  text-decoration: underline;
}
</style>
