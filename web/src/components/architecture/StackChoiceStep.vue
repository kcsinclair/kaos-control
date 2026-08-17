<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
// Milestone 3 (FR-6, FR-10): after an architecture is chosen (either path),
// lists its compatible stacks, language-ranked by the backend, with a
// recommended top pick plus the option to override with any other
// compatible stack. Shared by both the Guided and Browse paths.
import { computed, onMounted, ref, watch } from 'vue'
import { useArchitectureWizardStore } from '@/stores/architectureWizard'
import type { CatalogItem } from '@/types/api'

const props = defineProps<{
  project: string
}>()

const emit = defineEmits<{
  chosen: [item: CatalogItem]
}>()

const store = useArchitectureWizardStore()
const stacks = ref<CatalogItem[]>([])

const languageQuestion = computed(() => store.questions.find((q) => q.kind === 'language'))
const selectedLanguage = ref<string>(
  (languageQuestion.value && store.answerFor(languageQuestion.value.id)) || '',
)

async function load(): Promise<void> {
  stacks.value = await store.fetchStacks(props.project, selectedLanguage.value || undefined)
}

onMounted(load)
watch(selectedLanguage, load)
</script>

<template>
  <div class="stack-step">
    <div class="stack-header">
      <h2 class="stack-title">
        Compatible stacks for <span class="stack-arch-name">{{ store.chosenArchitecture?.title }}</span>
      </h2>
      <label v-if="languageQuestion" class="language-picker">
        Strongest language
        <select v-model="selectedLanguage" aria-label="Filter by strongest language">
          <option value="">Any</option>
          <option v-for="o in languageQuestion.options" :key="o.value" :value="o.value">
            {{ o.label }}
          </option>
        </select>
      </label>
    </div>

    <div v-if="store.loading" class="stack-state" role="status" aria-live="polite">
      Loading compatible stacks…
    </div>
    <div v-else-if="stacks.length === 0" class="stack-state empty">
      No compatible stacks found for this architecture yet.
    </div>

    <ul v-else class="stack-cards" role="list" aria-label="Compatible tech stacks">
      <li v-for="(s, i) in stacks" :key="s.path" class="stack-card">
        <button type="button" class="stack-card-btn" @click="emit('chosen', s)">
          <span v-if="i === 0" class="stack-recommended-badge">Recommended</span>
          <span class="stack-card-title">{{ s.title }}</span>
          <span class="stack-card-summary">{{ s.summary }}</span>
          <span v-if="s.labels.length" class="stack-card-labels">
            <span v-for="l in s.labels" :key="l" class="label-chip">{{ l }}</span>
          </span>
          <span class="stack-card-select">{{ i === 0 ? 'Confirm recommended stack' : 'Choose this stack' }}</span>
        </button>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.stack-step {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
.stack-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  flex-wrap: wrap;
}
.stack-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
}
.stack-arch-name { color: var(--color-accent); }
.language-picker {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}
.language-picker select {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-text);
}
.stack-state {
  padding: var(--space-4);
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}
.stack-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--space-4);
  list-style: none;
  margin: 0;
  padding: 0;
}
.stack-card-btn {
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
.stack-card-btn:hover,
.stack-card-btn:focus-visible {
  border-color: var(--color-accent);
}
.stack-recommended-badge {
  align-self: flex-start;
  padding: 1px 8px;
  background: var(--color-accent);
  color: #fff;
  border-radius: var(--radius-full);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
}
.stack-card-title {
  font-size: var(--text-base);
  font-weight: 700;
  color: var(--color-text);
}
.stack-card-summary {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  line-height: 1.5;
}
.stack-card-labels {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}
.label-chip {
  padding: 1px 8px;
  background: var(--color-border);
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text);
}
.stack-card-select {
  margin-top: auto;
  padding-top: var(--space-2);
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-accent);
}
</style>
