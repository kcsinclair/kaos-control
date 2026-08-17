<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script setup lang="ts">
// Milestone 3 (FR-4): the Browse vs Guided fork. Also reused in a compact
// form as the persistent "show me everything anyway" control surfaced
// throughout the Guided flow (FR-4) — same choice, presented inline.
withDefaults(
  defineProps<{
    variant?: 'full' | 'inline'
  }>(),
  { variant: 'full' },
)

const emit = defineEmits<{
  choose: [path: 'browse' | 'guided']
}>()
</script>

<template>
  <div v-if="variant === 'full'" class="path-choice">
    <h2 class="path-choice-title">How would you like to choose an architecture?</h2>
    <p class="path-choice-copy">
      Not sure which is right for you? Guided is the easy way — answer a few plain-language
      questions and we'll suggest the best fit. Prefer to see everything yourself? Browse the
      full catalog instead.
    </p>
    <div class="path-cards">
      <button type="button" class="path-card" @click="emit('choose', 'guided')">
        <span class="path-card-title">Guided</span>
        <span class="path-card-desc">
          Answer up to 10 short questions (every one skippable) and get a transparent
          recommendation with the reasoning behind it.
        </span>
      </button>
      <button type="button" class="path-card" @click="emit('choose', 'browse')">
        <span class="path-card-title">Browse</span>
        <span class="path-card-desc">
          See every candidate architecture and tech stack up front, with pros/cons, and pick
          directly.
        </span>
      </button>
    </div>
  </div>

  <div v-else class="path-choice-inline">
    <button type="button" class="show-everything-btn" @click="emit('choose', 'browse')">
      Show me everything anyway
    </button>
  </div>
</template>

<style scoped>
.path-choice {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.path-choice-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 700;
  color: var(--color-text);
}
.path-choice-copy {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  line-height: 1.6;
}
.path-cards {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-4);
}
.path-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-5);
  background: var(--color-bg, transparent);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  text-align: left;
  cursor: pointer;
}
.path-card:hover,
.path-card:focus-visible {
  border-color: var(--color-accent);
}
.path-card-title {
  font-size: var(--text-base);
  font-weight: 700;
  color: var(--color-text);
}
.path-card-desc {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  line-height: 1.5;
}
.path-choice-inline {
  display: flex;
  justify-content: flex-end;
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
