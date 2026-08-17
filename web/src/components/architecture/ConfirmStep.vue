<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
// Milestone 6 (FR-13, FR-14, NFR-1): the pre-write review. Nothing is
// written under lifecycle/architecture/ until "Confirm & write" is clicked.
// Builds the Q&A trail and architecture-breaking-requirements mapping the
// commit body needs from the store's questions/answers (kind: 'hard'
// questions whose chosen option carries hard: true).
import { computed } from 'vue'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import type { WizardBreakingReq, WizardQAPair } from '@/types/api'

const props = defineProps<{
  project: string
}>()

const emit = defineEmits<{
  committed: []
}>()

const store = useArchitectureWizardStore()

function optionFor(questionId: string, value: string) {
  return store.questions.find((q) => q.id === questionId)?.options?.find((o) => o.value === value)
}

const qa = computed<WizardQAPair[]>(() =>
  store.answers.map((a) => {
    const question = store.questions.find((q) => q.id === a.question_id)
    const option = optionFor(a.question_id, a.value)
    return {
      Question: question?.prompt ?? a.question_id,
      Answer: option?.label ?? a.value,
    }
  }),
)

const breakingRequirements = computed<WizardBreakingReq[]>(() => {
  const arch = store.chosenArchitecture
  if (!arch) return []
  return store.answers
    .map((a) => ({ answer: a, question: store.questions.find((q) => q.id === a.question_id) }))
    .filter(({ question }) => question?.kind === 'hard')
    .map(({ answer, question }) => {
      const option = optionFor(answer.question_id, answer.value)
      if (!option?.hard) return null
      const label = option.labels?.[0] ?? question!.id
      const satisfied = arch.labels.includes(label)
      return {
        Label: label,
        Requirement: question!.prompt,
        Mapping: satisfied
          ? `${arch.title} supports this.`
          : `${arch.title} is the closest available match — no exact match for this requirement.`,
      }
    })
    .filter((r): r is WizardBreakingReq => r !== null)
})

async function confirmAndWrite(): Promise<void> {
  const result = await store.commit(props.project, {
    breakingRequirements: breakingRequirements.value,
    qa: qa.value,
  })
  if (result) emit('committed')
}
</script>

<template>
  <div class="confirm-step">
    <h2 class="confirm-title">Review before writing</h2>
    <p class="confirm-copy">
      Nothing is written to the project until you confirm. Review your selection below.
    </p>

    <dl class="confirm-summary">
      <dt>Architecture</dt>
      <dd>{{ store.chosenArchitecture?.title }}</dd>
      <dt>Tech stack</dt>
      <dd>{{ store.chosenStack?.title }}</dd>
    </dl>

    <div v-if="breakingRequirements.length" class="confirm-section">
      <h3 class="confirm-section-title">Architecture-breaking requirements</h3>
      <table class="breaking-table">
        <thead>
          <tr>
            <th scope="col">Requirement</th>
            <th scope="col">How it's satisfied</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="b in breakingRequirements" :key="b.Label">
            <td>{{ b.Requirement }}</td>
            <td>{{ b.Mapping }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="confirm-section">
      <h3 class="confirm-section-title">Standards</h3>
      <p class="confirm-standards-note">
        Any standards already recorded under <code>lifecycle/architecture/standards/</code> will
        be linked from the architecture summary. None are seeded automatically by this wizard.
      </p>
    </div>

    <button
      type="button"
      class="btn-primary confirm-write-btn"
      :disabled="store.loading"
      @click="confirmAndWrite"
    >{{ store.loading ? 'Writing…' : 'Confirm & write' }}</button>
  </div>
</template>

<style scoped>
.confirm-step {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.confirm-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
}
.confirm-copy {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  line-height: 1.6;
}
.confirm-summary {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--space-1) var(--space-3);
  margin: 0;
  font-size: var(--text-sm);
}
.confirm-summary dt {
  font-weight: 600;
  color: var(--color-text);
}
.confirm-summary dd {
  margin: 0;
  color: var(--color-text-muted);
}
.confirm-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.confirm-section-title {
  margin: 0;
  font-size: var(--text-base);
  font-weight: 700;
  color: var(--color-text);
}
.breaking-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
.breaking-table th,
.breaking-table td {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  text-align: left;
}
.confirm-standards-note {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  line-height: 1.6;
}
.confirm-write-btn {
  align-self: flex-start;
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
</style>
