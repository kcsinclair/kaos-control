<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
// Milestone 5 (FR-9, FR-10, FR-11, NFR-4): transparent results — never a
// single black-box answer. Shows a summary of the answers given, then the
// best-fit architecture when the answers point to one, or an explicit
// "no exact match — closest fits" (OQ-2 dropped constraints) / "no clear
// match" state otherwise. Each candidate carries a visible "why", and an
// override expander falls back to Browse for any other catalog item (FR-4).
import { ref, computed, onMounted } from 'vue'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import type { CatalogItem } from '@/types/api'

const props = defineProps<{ project: string }>()

const emit = defineEmits<{
  chosen: [item: CatalogItem]
  'browse-anyway': []
}>()

const store = useArchitectureWizardStore()
const overrideExpanded = ref(false)

// Own the recommendation fetch rather than relying on the questions step
// having populated it. This makes the step robust to a page refresh, back
// navigation, or a race where it mounts before the prior step's fetch
// resolved (which showed a blank result until a manual refresh). Skip when
// results are already loaded to avoid a redundant call.
onMounted(() => {
  if (store.answers.length > 0 && store.recommendations.length === 0 && !store.loading) {
    void store.fetchRecommendations(props.project)
  }
})

// Every question the user went through, paired with the option label they
// picked (or "No preference" when skipped/unanswered) — so the recommendation
// is shown in the context of the answers that produced it.
const qaSummary = computed(() =>
  store.questions.map((q) => {
    const value = store.answerFor(q.id)
    const label =
      value === undefined
        ? 'No preference'
        : (q.options?.find((o) => o.value === value)?.label ?? value)
    return { id: q.id, prompt: q.prompt, answer: label, answered: value !== undefined }
  }),
)

const hasResults = computed(() => store.recommendations.length > 0)
// An exact match: the answers were satisfiable without relaxing any hard
// constraint. A relaxed result still has candidates, but they are the closest
// fits, not a true best fit.
const isExactMatch = computed(() => hasResults.value && store.droppedConstraints.length === 0)

const heading = computed(() => {
  if (!hasResults.value) return 'No clear match'
  return isExactMatch.value ? 'Your best-fit architecture' : 'No exact match — here are the closest fits'
})
</script>

<template>
  <div class="recommend-step">
    <!-- Summary of the answers that produced this recommendation. -->
    <details class="qa-summary" open>
      <summary class="qa-summary-title">Your answers</summary>
      <dl class="qa-list">
        <template v-for="qa in qaSummary" :key="qa.id">
          <dt class="qa-prompt">{{ qa.prompt }}</dt>
          <dd class="qa-answer" :class="{ 'qa-answer--none': !qa.answered }">{{ qa.answer }}</dd>
        </template>
      </dl>
    </details>

    <div v-if="store.loading" class="recommend-loading" role="status">
      Finding your best-fit architecture…
    </div>

    <template v-else>
    <h2 class="recommend-title">{{ heading }}</h2>

    <div
      v-if="store.droppedConstraints.length > 0"
      class="dropped-banner"
      role="status"
    >
      No architecture matched every answer, so we relaxed these to find the closest fits:
      <ul>
        <li v-for="c in store.droppedConstraints" :key="c">{{ c }}</li>
      </ul>
    </div>

    <!-- No candidates at all (e.g. an empty catalog): tell the user plainly
         and point them at Browse rather than showing a "best fit" with nothing
         under it. -->
    <div v-if="!hasResults" class="no-match">
      <p>
        Your answers didn't point to a single architecture. Browse the full catalog to choose one
        directly.
      </p>
      <button type="button" class="show-everything-btn" @click="emit('browse-anyway')">
        Browse the full catalog
      </button>
    </div>

    <ul v-else class="recommend-cards" role="list" aria-label="Recommended architectures">
      <li
        v-for="(r, i) in store.recommendations"
        :key="r.item.path"
        class="recommend-card"
        :class="{ 'recommend-card--best': i === 0 }"
      >
        <button type="button" class="recommend-card-btn" @click="emit('chosen', r.item)">
          <span v-if="i === 0" class="recommend-badge">
            {{ isExactMatch ? 'Best fit' : 'Closest match' }}
          </span>
          <span class="recommend-card-title">{{ r.item.title }}</span>
          <span class="recommend-card-summary">{{ r.item.summary }}</span>
          <ul class="recommend-why" aria-label="Why this fits">
            <li v-for="(w, wi) in r.why" :key="wi">{{ w }}</li>
          </ul>
          <span class="recommend-card-select">Choose this architecture</span>
        </button>
      </li>
    </ul>

    <div class="override-section">
      <button
        type="button"
        class="override-toggle"
        :aria-expanded="overrideExpanded"
        @click="overrideExpanded = !overrideExpanded"
      >
        None of these quite right? Override with any other candidate
      </button>
      <div v-if="overrideExpanded" class="override-body">
        <button type="button" class="show-everything-btn" @click="emit('browse-anyway')">
          Browse the full catalog
        </button>
      </div>
    </div>
    </template>
  </div>
</template>

<style scoped>
.recommend-step {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.qa-summary {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4);
  background: var(--color-surface);
}
.qa-summary-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  cursor: pointer;
}
.qa-list {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: var(--space-1) var(--space-4);
  margin: var(--space-3) 0 0;
}
.qa-prompt {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  margin: 0;
}
.qa-answer {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
  text-align: right;
}
.qa-answer--none {
  font-weight: 400;
  font-style: italic;
  color: var(--color-text-muted);
}
.recommend-loading {
  padding: var(--space-4);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.recommend-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
}
.dropped-banner {
  padding: var(--space-3) var(--space-4);
  background: #fef3c7;
  border: 1px solid #fcd34d;
  border-radius: var(--radius-md);
  color: #92400e;
  font-size: var(--text-sm);
  line-height: 1.6;
}
.dropped-banner ul {
  margin: var(--space-1) 0 0;
  padding-left: 1.2em;
}
.no-match {
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  line-height: 1.6;
}
.no-match p {
  margin: 0 0 var(--space-3);
}
.recommend-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--space-4);
  list-style: none;
  margin: 0;
  padding: 0;
}
.recommend-card--best .recommend-card-btn {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 1px var(--color-accent);
}
.recommend-card-btn {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  width: 100%;
  height: 100%;
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  text-align: left;
  cursor: pointer;
}
.recommend-card-btn:hover,
.recommend-card-btn:focus-visible {
  border-color: var(--color-accent);
}
.recommend-badge {
  align-self: flex-start;
  padding: 2px var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--color-accent);
  color: #fff;
  font-size: var(--text-xs);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
.recommend-card-title {
  font-size: var(--text-base);
  font-weight: 700;
  color: var(--color-text);
}
.recommend-card-summary {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  line-height: 1.5;
}
.recommend-why {
  margin: 0;
  padding-left: 1.1em;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.recommend-card-select {
  margin-top: auto;
  padding-top: var(--space-2);
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-accent);
}
.override-section {
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-4);
}
.override-toggle {
  background: none;
  border: none;
  padding: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  cursor: pointer;
  text-decoration: underline;
}
.override-body {
  margin-top: var(--space-3);
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
</style>
